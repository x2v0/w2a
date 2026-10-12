package apk

import "github.com/x2v0/w2a/go/internal/arsc"

// ListLocales returns locale tags present in resources.arsc.
func (a *APK) ListLocales() ([]arsc.LocaleInfo, error) {
	t, err := a.LoadResources()
	if err != nil {
		return nil, err
	}
	return t.ListLocales(), nil
}

// StripLocale zeroes locale-tagged Type configs matching tag ("ar", "ru", "*").
func (a *APK) StripLocale(tag string) (int, error) {
	t, err := a.LoadResources()
	if err != nil {
		return 0, err
	}
	n := t.StripLocale(tag)
	if err := a.SaveResources(t); err != nil {
		return n, err
	}
	return n, nil
}

// KeepLocales keeps only the listed locales; others zeroed.
func (a *APK) KeepLocales(keep []string) (int, error) {
	t, err := a.LoadResources()
	if err != nil {
		return 0, err
	}
	n := t.KeepLocales(keep)
	if err := a.SaveResources(t); err != nil {
		return n, err
	}
	return n, nil
}

// StripScript replaces strings whose dominant script matches name.
func (a *APK) StripScript(name, repl string) (int, error) {
	t, err := a.LoadResources()
	if err != nil {
		return 0, err
	}
	n := t.StripScript(name, repl)
	if err := a.SaveResources(t); err != nil {
		return n, err
	}
	return n, nil
}
