//go:build windows

package companion

import "golang.org/x/sys/windows/registry"

func platformGameRoots() []string {
	out := []string{}
	for _, path := range []string{`SOFTWARE\Blizzard Entertainment\World of Warcraft`, `SOFTWARE\WOW6432Node\Blizzard Entertainment\World of Warcraft`} {
		key, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		for _, name := range []string{"InstallPath", "InstallLocation"} {
			if value, _, err := key.GetStringValue(name); err == nil && value != "" {
				out = append(out, value)
			}
		}
		key.Close()
	}
	return out
}
