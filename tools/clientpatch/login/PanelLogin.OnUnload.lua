function PanelLogin:OnUnload()
  if self.passwordInputField then
    self.passwordInputField.text = ""
  end
  super(PanelLogin):OnUnload()
  MC.AudioManager:PlayEvent("music_login_stop")
  MC.GameWorld.InputKeyMgr:UnRegisterClick(CS.UnityEngine.KeyCode.Return, bind(self, self.OnReturn))
  MC.GameWorld.InputKeyMgr:UnRegisterClick(CS.UnityEngine.KeyCode.KeypadEnter, bind(self, self.OnReturn))
  ServerListHelper.Instance():Stop()
end
