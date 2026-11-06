package companion

import (
	lua "github.com/yuin/gopher-lua"
	"os"
	"path/filepath"
	"testing"
)

func TestOCRLabCorrelatesReplyAndNeverResendsOnFrame(t *testing.T) {
	vm := lua.NewState()
	defer vm.Close()
	mock := `
 SlashCmdList={};UIParent={};WispOCRSession='23456789';NOW=100;GUID='Player-1234-ABCDEF';LOADS=0;FRAMES={}
 local methods={}
 local meta={__index=function(t,k) return methods[k] or function()end end}
 function CreateFrame(kind,name) local f=setmetatable({scripts={},name=name},meta);FRAMES[#FRAMES+1]=f;return f end
 function methods:SetScript(k,f)self.scripts[k]=f end
 function methods:CreateFontString()return setmetatable({},meta)end
 function methods:CreateTexture()return setmetatable({},meta)end
 function methods:SetText(s)self.text=s end
 function methods:GetText()return self.text end
 function methods:Disable()self.disabled=true end;function methods:Enable()self.disabled=false end
 function print()end;function time()return 1793960000 end;function GetTime()return NOW end;function UnitGUID()return GUID end
 C_AddOns={IsAddOnLoaded=function()return false end,LoadAddOn=function()LOADS=LOADS+1;return true end}
 `
	if err := vm.DoString(mock); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("../../apps/ocr-lab/WispOCR/WispOCR.lua")
	if err != nil {
		t.Fatal(err)
	}
	if err = vm.DoString(string(b)); err != nil {
		t.Fatal(err)
	}
	if err = vm.DoString(`
 local panel,edit=FRAMES[1],FRAMES[2]
 assert(LOADS==0)
 edit:SetText('Where should I go next?');edit.scripts.OnEnterPressed(edit);assert(edit.disabled)
 edit.scripts.OnEnterPressed(edit);assert(LOADS==0)
 panel.scripts.OnUpdate(panel,.7);assert(LOADS==1)
 WispOCRReceive('wrong',GUID,'done','4869');assert(edit.disabled)
 WispOCRReceive('234567891793960000000001','Player-other','done','4869');assert(edit.disabled)
 WispOCRReceive('234567891793960000000001',GUID,'thinking','');assert(edit.disabled)
 WispOCRReceive('234567891793960000000001',GUID,'done','4869');assert(not edit.disabled)
 panel.scripts.OnUpdate(panel,.7);assert(LOADS==1)
 edit:SetText('Next question');edit.scripts.OnEnterPressed(edit);assert(edit.disabled)
 GUID='Player-other';panel.scripts.OnUpdate(panel,.7);assert(not edit.disabled and LOADS==1)
 `); err != nil {
		t.Fatal(err)
	}
}

func TestOCRSessionIsExplicitPrivateAndRefusesOverwrite(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "player.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.DB.Close()
	server := &Server{Store: s, Host: "127.0.0.1:8788", Token: "test-secret"}
	path := filepath.Join(t.TempDir(), "session.json")
	if err = server.WriteOCRSession(path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatal("session not owner-only")
	}
	if server.WriteOCRSession(path) == nil {
		t.Fatal("overwrote existing session")
	}
	link := filepath.Join(t.TempDir(), "symlink")
	if err = os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if server.WriteOCRSession(link) == nil {
		t.Fatal("followed symlink")
	}
}
