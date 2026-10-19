package target

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/x2v0/w2a/go/internal/apkutil"
	"github.com/x2v0/w2a/go/internal/config"
)

// CustomTarget builds an APK from plain Java sources + jniLibs.
// No gradle, no kotlinc, no gomobile: javac -> jar -> d8 -> aapt2 -> inject.
type CustomTarget struct{}

func (t *CustomTarget) Name() string { return "custom" }

func (t *CustomTarget) Prepare(cfg *config.Config, templatePath, outPath string) error {
	if cfg.Custom == nil {
		return fmt.Errorf("custom: missing 'custom' section")
	}
	cs := cfg.Custom
	sdkRoot := os.Getenv("ANDROID_HOME")
	if sdkRoot == "" {
		sdkRoot = os.Getenv("ANDROID_SDK_ROOT")
	}
	if sdkRoot == "" {
		return fmt.Errorf("custom: ANDROID_HOME not set")
	}

	minAPI := cs.MinAPI
	if minAPI == 0 {
		minAPI = 26
	}
	targetAPI := cs.TargetAPI
	if targetAPI == 0 {
		targetAPI = 34
	}
	javaRelease := cs.JavaRelease
	if javaRelease == "" {
		javaRelease = "17"
	}

	fmt.Printf("custom: SDK=%s minAPI=%d targetAPI=%d javaRelease=%s\n",
		sdkRoot, minAPI, targetAPI, javaRelease)

	androidJar, err := apkutil.FindAndroidJar(sdkRoot)
	if err != nil {
		return fmt.Errorf("custom: %w", err)
	}
	fmt.Printf("custom: android.jar=%s\n", androidJar)

	bt, err := apkutil.FindNewestBuildTool(sdkRoot)
	if err != nil {
		return fmt.Errorf("custom: %w", err)
	}
	d8Jar := filepath.Join(bt, "lib", "d8.jar")
	if _, err := os.Stat(d8Jar); err != nil {
		return fmt.Errorf("custom: d8.jar not found: %s", d8Jar)
	}
	fmt.Printf("custom: d8.jar=%s\n", d8Jar)

	aapt2, err := apkutil.FindAapt2(sdkRoot)
	if err != nil {
		return fmt.Errorf("custom: %w", err)
	}
	fmt.Printf("custom: aapt2=%s\n", aapt2)

	tmp, err := os.MkdirTemp("", "wtacustom-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	// ── 1. collect Java sources ─────────────────────────────
	sources, err := apkutil.CollectJavaSources(cs.Sources)
	if err != nil {
		return fmt.Errorf("custom: sources: %w", err)
	}
	if len(sources) == 0 {
		return fmt.Errorf("custom: no .java files under %v", cs.Sources)
	}
	fmt.Printf("custom: %d .java files\n", len(sources))

	// ── 2. javac ────────────────────────────────────────────
	classesDir := filepath.Join(tmp, "classes")
	classpath := androidJar
	for _, extra := range cs.Classpath {
		classpath += string(os.PathListSeparator) + cfg.Resolve(extra)
	}
	if err := apkutil.Javac(javaRelease, classpath, classesDir, sources); err != nil {
		return fmt.Errorf("custom: %w", err)
	}
	nClasses, _ := countFiles(classesDir, ".class")
	fmt.Printf("custom: javac -> %d .class\n", nClasses)

	// ── 3. jar + d8 ─────────────────────────────────────────
	classesJar := filepath.Join(tmp, "classes.jar")
	if err := apkutil.JarClasses(classesDir, classesJar); err != nil {
		return fmt.Errorf("custom: %w", err)
	}
	dexDir := filepath.Join(tmp, "dex")
	// d8 is invoked via `java -cp d8.jar com.android.tools.r8.D8`
	// Wrap in a tiny shell adapter, or call java directly.
	// We call java directly using a fake D8Binary=java and rebuild args.
	if err := runD8ThroughJava(d8Jar, androidJar, classesJar, dexDir, minAPI); err != nil {
		return fmt.Errorf("custom: %w", err)
	}
	dexPath := filepath.Join(dexDir, "classes.dex")
	if _, err := os.Stat(dexPath); err != nil {
		return fmt.Errorf("custom: d8 produced no classes.dex: %w", err)
	}
	dexInfo, _ := os.Stat(dexPath)
	fmt.Printf("custom: d8 -> classes.dex (%d bytes)\n", dexInfo.Size())

	// ── 4. aapt2 link (base APK from manifest + res) ────────
	var resZip string
	if cs.Resources != "" {
		resDir := cfg.Resolve(cs.Resources)
		if _, err := os.Stat(resDir); err == nil {
			resZip = filepath.Join(tmp, "res.zip")
			if err := apkutil.Aapt2Compile(aapt2, resDir, resZip); err != nil {
				return fmt.Errorf("custom: %w", err)
			}
			fmt.Printf("custom: aapt2 compile -> res.zip\n")
		}
	}
	baseAPK := filepath.Join(tmp, "base.apk")
	manifestPath := cfg.Resolve(cs.Manifest)
	if err := apkutil.Aapt2Link(aapt2, androidJar, manifestPath, resZip,
		baseAPK, minAPI, targetAPI); err != nil {
		return fmt.Errorf("custom: %w", err)
	}
	baseInfo, _ := os.Stat(baseAPK)
	fmt.Printf("custom: aapt2 link -> base.apk (%d bytes)\n", baseInfo.Size())

	// ── 5. inject dex + jniLibs ─────────────────────────────
	inject := map[string]string{"classes.dex": dexPath}
	if len(cs.JNILibs) > 0 {
		absJNIDirs := make([]string, 0, len(cs.JNILibs))
		for _, d := range cs.JNILibs {
			absJNIDirs = append(absJNIDirs, cfg.Resolve(d))
		}
		libs, err := apkutil.CollectJNILibs(absJNIDirs, cs.ABIs)
		if err != nil {
			return fmt.Errorf("custom: %w", err)
		}
		for k, v := range libs {
			inject[k] = v
		}
		fmt.Printf("custom: injecting %d jniLibs\n", len(libs))
	}
	if err := apkutil.InjectIntoAPK(baseAPK, inject); err != nil {
		return fmt.Errorf("custom: %w", err)
	}

	// ── 6. copy to outPath ──────────────────────────────────
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	if err := copyFileSimple(baseAPK, outPath); err != nil {
		return err
	}
	fmt.Printf("custom: stage1 -> %s\n", outPath)
	return nil
}

// runD8ThroughJava invokes `java -cp d8.jar com.android.tools.r8.D8 ...`
func runD8ThroughJava(d8Jar, androidJar, inputJar, outDir string, minAPI int) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	args := []string{
		"-Xmx1g", "-cp", d8Jar, "com.android.tools.r8.D8",
		"--release",
		"--min-api", fmt.Sprintf("%d", minAPI),
		"--lib", androidJar,
		"--output", outDir,
		inputJar,
	}
	cmd := exec.Command("java", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("d8: %v\n%s", err, string(out))
	}
	return nil
}

func countFiles(dir, suffix string) (int, error) {
	n := 0
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(p) == suffix {
			n++
		}
		return nil
	})
	return n, err
}

func copyFileSimple(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	buf := make([]byte, 64*1024)
	for {
		n, err := in.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return werr
			}
		}
		if err != nil {
			break
		}
	}
	return nil
}
