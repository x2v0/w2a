package musl

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Config struct {
	MuslVersion string
	TarballURL  string
	TarballPath string
	WorkDir     string
	OutputDir   string
	NDKPath     string
	PatchPath   string
	ShimPath    string
	ABIs        []string
}

type target struct {
	triple string
	rtLib  string
}

var targetsByABI = map[string]target{
	"arm64-v8a": {"aarch64-linux-musl", "libclang_rt.builtins-aarch64-android.a"},
	"x86_64":    {"x86_64-linux-musl", "libclang_rt.builtins-x86_64-android.a"},
}

func Build(cfg Config) error {
	if cfg.MuslVersion == "" {
		cfg.MuslVersion = "1.2.5"
	}
	if cfg.TarballURL == "" {
		cfg.TarballURL = "https://musl.libc.org/releases/musl-" + cfg.MuslVersion + ".tar.gz"
	}
	if cfg.WorkDir == "" {
		return fmt.Errorf("WorkDir required")
	}
	if cfg.OutputDir == "" {
		return fmt.Errorf("OutputDir required")
	}
	if err := os.MkdirAll(cfg.WorkDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(cfg.OutputDir, 0o755); err != nil {
		return err
	}

	tarball := cfg.TarballPath
	if tarball == "" {
		tarball = filepath.Join(cfg.WorkDir, "musl-"+cfg.MuslVersion+".tar.gz")
		if _, err := os.Stat(tarball); err != nil {
			if err := download(cfg.TarballURL, tarball); err != nil {
				return fmt.Errorf("download: %w", err)
			}
		}
	}

	src := filepath.Join(cfg.WorkDir, "musl-"+cfg.MuslVersion)
	if err := os.RemoveAll(src); err != nil {
		return err
	}
	if err := extractTarGz(tarball, cfg.WorkDir); err != nil {
		return fmt.Errorf("extract: %w", err)
	}

	if cfg.PatchPath != "" {
		data, err := os.ReadFile(cfg.PatchPath)
		if err != nil {
			return fmt.Errorf("read patch: %w", err)
		}
		if err := ApplyPatch(cfg.WorkDir, data, 1); err != nil {
			return fmt.Errorf("apply patch: %w", err)
		}
	}

	if cfg.ShimPath != "" {
		dir := filepath.Join(src, "src", "math", "x86_64")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		data, err := os.ReadFile(cfg.ShimPath)
		if err != nil {
			return fmt.Errorf("read shim: %w", err)
		}
		if err := os.WriteFile(filepath.Join(dir, filepath.Base(cfg.ShimPath)), data, 0o644); err != nil {
			return err
		}
	}

	tc, err := toolchainDir(cfg.NDKPath)
	if err != nil {
		return err
	}
	clang := withExe(filepath.Join(tc, "bin", "clang"))
	strip := withExe(filepath.Join(tc, "bin", "llvm-strip"))
	if _, err := os.Stat(clang); err != nil {
		return fmt.Errorf("clang not found: %s", clang)
	}

	jobs := fmt.Sprintf("%d", runtime.NumCPU())
	for _, abi := range cfg.ABIs {
		t, ok := targetsByABI[abi]
		if !ok {
			return fmt.Errorf("unsupported abi: %s", abi)
		}
		rt, err := findCompilerRt(tc, t.rtLib)
		if err != nil {
			return err
		}
		buildDir := filepath.Join(cfg.WorkDir, "build-"+abi)
		if err := os.RemoveAll(buildDir); err != nil {
			return err
		}
		if err := os.MkdirAll(buildDir, 0o755); err != nil {
			return err
		}

		env := append(os.Environ(),
			"CC="+clang+" --target="+t.triple,
			"CROSS_COMPILE="+filepath.Join(tc, "bin", "llvm-"),
			"CFLAGS=-O2",
			"LDFLAGS=-Wl,-z,max-page-size=16384",
		)

		configure := filepath.Join("..", "musl-"+cfg.MuslVersion, "configure")
		if err := runCmd(buildDir, env, configure, "--target="+t.triple); err != nil {
			return fmt.Errorf("configure %s: %w", abi, err)
		}
		if err := runCmd(buildDir, env, "make", "-j"+jobs, "LIBCC="+rt); err != nil {
			return fmt.Errorf("make %s: %w", abi, err)
		}

		outDir := filepath.Join(cfg.OutputDir, abi)
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return err
		}
		out := filepath.Join(outDir, "libmusl-linker.so")
		if err := runCmd(buildDir, env, strip,
			filepath.Join(buildDir, "lib", "libc.so"),
			"-o", out); err != nil {
			return fmt.Errorf("strip %s: %w", abi, err)
		}
		_ = os.Chmod(out, 0o755)
		fmt.Printf("built %s\n", out)
	}
	return nil
}

func download(url, dst string) error {
	fmt.Printf("downloading %s\n", url)
	client := &http.Client{Timeout: 3 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	tmp := dst + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		return err
	}
	f.Close()
	return os.Rename(tmp, dst)
}

func extractTarGz(archive, destDir string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	clean := filepath.Clean(destDir)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		dst := filepath.Join(destDir, filepath.Clean(hdr.Name))
		if !strings.HasPrefix(dst, clean+string(os.PathSeparator)) && dst != clean {
			continue
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(dst, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return err
			}
			out, err := os.Create(dst)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			out.Close()
			_ = os.Chmod(dst, os.FileMode(hdr.Mode)&0o777)
		}
	}
	return nil
}

func findCompilerRt(tc, name string) (string, error) {
	base := filepath.Join(tc, "lib", "clang")
	entries, err := os.ReadDir(base)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := filepath.Join(base, e.Name(), "lib", "linux", name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("compiler-rt %s not found under %s", name, base)
}

func runCmd(cwd string, env []string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = cwd
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
