function PanelLogin:OnResponsePharosSpeedup(speedupIp, speedupPort)
  if self._curSpeedupServer == nil then
    return
  end
  if Switch.FORCE_DEBUG_LOGIN then
    local userName = string.trim(self.userNameInputField.text)
    local password = self.passwordInputField.text
    local password = self.passwordInputField and self.passwordInputField.text or ""
    if #userName == 0 or #userName > 64 or string.find(userName, "[%z\r\n]") then
      self.userNameInputField:ActivateInputField()
      self._curSpeedupServer = nil
      MC.UIManager:ClosePanel("PanelWaiting")
      return
    end
    if #password == 0 or #password > 256 or string.find(password, "%z") then
      if self.passwordInputField then
        self.passwordInputField:ActivateInputField()
      end
      self._curSpeedupServer = nil
      MC.UIManager:ClosePanel("PanelWaiting")
      return
    end
  end
  local ip = speedupIp or self._curSpeedupServer.ip
  local port = speedupPort or self._curSpeedupServer.port
  local serverName = self._curSpeedupServer.serverName
  GameConfig.ServerUrl = ip .. ":" .. tostring(port)
  local serverInfo = ServerInfo.Instance()
  local switchServer = serverName ~= serverInfo:GetServerName()
  if switchServer then
    self._disconnectForSwitchServer = true
    ServerClient.Instance():Disconnect()
    self._disconnectForSwitchServer = false
  end
  serverInfo:SetServerAddr(ip, port)
  serverInfo:SetOriginServerAddr(self._curSpeedupServer.ip, self._curSpeedupServer.port)
  serverInfo:SetIsIpv6(not isipv4(ip))
  serverInfo:SetServerName(serverName)
  serverInfo:SetServer(self._curSpeedupServer.server)
  serverInfo:OnTryLogin()
  self._curSpeedupServer = nil
  local fpsPanel = MC.UIManager:GetPanel("PanelFPS")
  if fpsPanel then
    fpsPanel:SetServerInfo(serverName)
  end
  PlayerPrefs.SetString("LastLoginServerName", serverName)
  PlayerPrefs.Save()
  self._noConnectFailedError = false
  if Switch.FORCE_DEBUG_LOGIN then
    local userName = string.trim(self.userNameInputField.text)
    local password = self.passwordInputField.text
    if userName == "" then
      return
    end
    local loginEntity = ServerClient.Instance():GetLoginEntity()
    if not switchServer and loginEntity then
      ServerClient.Instance():UpdateAccountInfo(userName, password)
      loginEntity:Login(userName, password)
    else
      ServerClient.Instance():ConnectServer(userName, password)
    end
    PlayerPrefs.SetString("LoginUserName", userName)
    PlayerPrefs.Save()
  else
    local loginEntity = ServerClient.Instance():GetLoginEntity()
    local sauthJson = MC.UniSDKHandler:GetSauthJson()
    if sauthJson and self:CheckLoginTokenValid() then
      if not switchServer and loginEntity then
        loginEntity:LoginPlatform(sauthJson)
      else
        ServerClient.Instance():ConnectServer("", "")
      end
    else
      MC.UIManager:ClosePanel("PanelWaiting")
      MC.UniSDKHandler:DoChannelLogin()
    end
  end
end
