package companion

import (
	lua "github.com/yuin/gopher-lua"
	"os"
	"testing"
)

func TestCompanionPanelModesResizeDraftAndPendingIsolation(t *testing.T) {
	vm := lua.NewState()
	defer vm.Close()
	if err := vm.DoString(`
 UIParent={};ChatFontNormal={};local methods={};local meta={__index=function(t,k)return methods[k] or function()end end}
 function CreateFrame(kind,name,parent) return setmetatable({scripts={},w=0,h=0,shown=true,lines={}},meta) end
 function methods:CreateFontString()return CreateFrame()end;function methods:CreateTexture()return CreateFrame()end
 function methods:SetPassword(v)self.password=v end
 function methods:SetFont(path,size,flags) assert(type(flags)=='string','Native font flags required') end
 function methods:SetScript(k,v)self.scripts[k]=v end
 function methods:SetSize(w,h)self.w=w;self.h=h;if self.scripts.OnSizeChanged then self.scripts.OnSizeChanged(self,w,h)end end
 function methods:SetWidth(w)self.w=w end;function methods:SetHeight(h)self.h=h end
 function methods:GetWidth()return self.w end;function methods:GetHeight()return self.h end
 function methods:GetFrameLevel()return 1 end;function methods:StartMoving()self.moving=true end
 function methods:SetShown(b)self.shown=b end;function methods:Show()self.shown=true end;function methods:Hide()self.shown=false end
 function methods:SetText(s)self.text=s;if self.scripts.OnTextChanged then self.scripts.OnTextChanged(self)end end
 function methods:GetText()return self.text end
 function methods:Clear()self.lines={} end;function methods:AddMessage(s)table.insert(self.lines,s)end
 function methods:SetEnabled(v)self.disabled=not v end;function methods:SetChecked(v)self.checked=v end;function methods:GetChecked()return self.checked end
 function methods:Disable()self.disabled=true end;function methods:Enable()self.disabled=false end
 function methods:StartSizing()self.sizing=true end;function methods:StopMovingOrSizing()self.sizing=false;self.moving=false end
 `); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile("../../apps/addon/Jornia/Companion.lua")
	if err != nil {
		t.Fatal(err)
	}
	if err = vm.DoString(string(source)); err != nil {
		t.Fatal(err)
	}
	if err = vm.DoString(`
 local UI=WispJournalUI;local a={id='one',name='One',level=16,zone='Ashenvale'};local b={id='two',name='Two',level=2,zone='Durotar'}
 UI.Show(a,{});local c=UI.Controls
 c.Input:SetText('A draft');UI.AddMessage('assistant','Welcome');UI.SetMode('compact')
 assert(c.Window.w==420 and c.Window.h==340 and c.Input:GetText()=='A draft' and c.Chat.shown and not c.Journey.shown)
 c.Window:SetSize(50,20);assert(c.Window.w==340 and c.Window.h==260 and c.Input.w>0)
 c.Resize.scripts.OnMouseDown(c.Resize,'LeftButton');assert(c.Window.sizing);c.Resize.scripts.OnMouseUp();assert(not c.Window.sizing)
 UI.DragHandle.scripts.OnMouseDown(UI.DragHandle,'LeftButton');assert(c.Window.moving);UI.DragHandle.scripts.OnMouseUp();assert(not c.Window.moving)
 UI.SetMode('full');assert(c.Window.w==860 and c.Input:GetText()=='A draft')
 UI.SetMode('compact');assert(c.Window.w==340 and #c.Messages.lines==1)
 UI.Show(b,{});assert(c.Input:GetText()=='' and #c.Messages.lines==0);c.Input:SetText('Other draft')
 UI.Show(a,{});assert(c.Input:GetText()=='A draft' and #c.Messages.lines==1)
 c.Input.scripts.OnEnterPressed();assert(c.Input:GetText()=='A draft')
 local sends=0;UI.OnSend=function(q,owner)sends=sends+1;assert(owner=='one');return 'op1' end
 c.Input.scripts.OnEnterPressed();assert(sends==1 and c.Input.disabled and c.Input:GetText()=='' and c.Status.text=='Sending your message…')
 UI.SetProgress('op1','one','Jornia is thinking…')
 UI.SetMode('full');UI.SetMode('compact');c.Input.scripts.OnEnterPressed();assert(sends==1 and c.Status.text=='Jornia is thinking…')
 assert(not UI.Resolve('wrong','one','Wrong'));assert(not UI.Resolve('op1','two','Wrong'))
 UI.Show(b,{});assert(c.Input.disabled and c.Input:GetText()=='Other draft' and #c.Messages.lines==0)
 assert(UI.Resolve('op1','one','Reply',false));assert(#c.Messages.lines==0 and not c.Input.disabled)
 UI.Show(a,{});assert(c.Input:GetText()=='' and #c.Messages.lines==3)
 c.Input:SetText('Keep on failure');c.Input.scripts.OnEnterPressed();assert(c.Input:GetText()=='');UI.Resolve('op1','one','error',true)
 assert(c.Input:GetText()=='Keep on failure' and not c.Input.disabled)
 UI.ShowJourney();assert(UI.GetMode()=='full' and c.Journey.shown)
 local before=#c.Messages.lines;UI.ShowConnection();assert(UI.connectionInput.password)
 local attempts=0;UI.OnConnect=function(code,owner)attempts=attempts+1;assert(code==string.rep('ab',32) and owner=='one');return 'connect1' end
 UI.connectionInput:SetText(string.rep('ab',32));UI.connectionInput.scripts.OnEnterPressed();assert(UI.connectionInput:GetText()=='' and attempts==1)
 UI.connectionInput.scripts.OnEnterPressed();assert(attempts==1 and #c.Messages.lines==before)
 assert(not UI.ResolveConnection('wrong','one','Wrong',false));assert(UI.ResolveConnection('connect1','one','Connected',false));assert(#c.Messages.lines==before and not c.Input.disabled)

 `); err != nil {
		t.Fatal(err)
	}
}

func TestNativePanelBindingsReadJournalWithoutRecordingOnInstall(t *testing.T) {
	vm := lua.NewState()
	defer vm.Close()
	if err := vm.DoString(`
 local calls={};CALLS=calls;SlashCmdList={WISP=function(value)table.insert(calls,value)end}
 function print()end;function UnitGUID()return 'Player-one' end;function UnitName()return 'One','Surname' end;function GetRealmName()return 'Realm' end;function UnitLevel()return 16 end;function GetZoneText()return 'Ashenvale' end
 WispDB={characters={['Player-one']={events={{kind='level_up',level=16}}}}}
 WispJournalUI={shown=false,Show=function(p,e)PERSON=p;EVENTS=e;WispJournalUI.shown=true end,IsShown=function()return WispJournalUI.shown end,Hide=function()WispJournalUI.shown=false end,SetMode=function(m)MODE=m end,ShowChat=function()TAB='chat' end,ShowJourney=function()TAB='journey' end}
 function CreateFrame()return {RegisterEvent=function()end,SetScript=function(_,_,f)EVENT=f end}end
 `); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile("../../apps/addon/Jornia/Bindings.lua")
	if err != nil {
		t.Fatal(err)
	}
	if err = vm.DoString(string(source)); err != nil {
		t.Fatal(err)
	}
	if err = vm.DoString(`
 assert(#CALLS==0 and not WispJournalUI.shown);EVENT();assert(PERSON==nil)
 SlashCmdList.WISP('');assert(PERSON.name=='One Surname' and PERSON.id=='Player-one' and PERSON.level==16 and #EVENTS==1 and #CALLS==0)
 SlashCmdList.WISP('compact');assert(MODE=='compact')
 SlashCmdList.WISP('journey');assert(MODE=='full' and TAB=='journey')
 WispJournalUI.OnNote('A note');assert(CALLS[1]=='note A note')
 WispJournalUI.OnPause();assert(CALLS[2]=='stop')
 SlashCmdList.WISP('export');assert(CALLS[3]=='export')
 SlashCmdList.WISP('');assert(not WispJournalUI.shown)
 SlashCmdList.WISPMINI();assert(MODE=='compact')
 `); err != nil {
		t.Fatal(err)
	}
}
