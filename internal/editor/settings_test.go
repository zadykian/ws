package editor

import "testing"

func TestTheSettingsLoad(t *testing.T) {
	settings, err := LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if settings.Font == "" || settings.Theme.Dark == "" {
		t.Errorf("settings %+v", settings)
	}
}

func TestTheFileTypesLoad(t *testing.T) {
	types, err := LoadFileTypes()
	if err != nil || len(types) == 0 {
		t.Fatalf("%v, %v", types, err)
	}
}

func TestTheCatalogueLoads(t *testing.T) {
	servers, err := LoadServers()
	if err != nil || len(servers) == 0 {
		t.Fatalf("%v, %v", servers, err)
	}
}
