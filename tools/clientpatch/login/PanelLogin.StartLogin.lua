function PanelLogin:StartLogin()
  if not Switch.FORCE_DEBUG_LOGIN then
    local sauthJson = MC.UniSDKHandler:GetSauthJson()
    if not sauthJson then
      MC.UniSDKHandler:DoChannelLogin()
    end
  end
  MC.AudioManager:PlayEvent("music_login_play")
end
