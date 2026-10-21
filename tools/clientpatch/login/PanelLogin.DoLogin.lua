function PanelLogin:DoLogin()

  if Switch.FORCE_DEBUG_LOGIN then
    local userName = string.trim(self.userNameInputField.text)
    local password = self.passwordInputField and self.passwordInputField.text or ""
    if #userName == 0 or #userName > 64 or string.find(userName, "[%z\r\n]") then
      self.userNameInputField:ActivateInputField()
      return
    end
    if #password == 0 or #password > 256 or string.find(password, "%z") then
      if self.passwordInputField then
        self.passwordInputField:ActivateInputField()
      end
      return
    end
  end
  if self:CheckFastClick(1) then
    return
  end
  self:MarkClickTime()
  if not self:FilterLowDevice() then
    return
  end
  if self._serverListHelper == nil then
    return
  end
  local servers = self._servers
  if not servers or #servers == 0 then
    self._serverListHelper:RefreshServerList()
    return
  end
  local index = self._serverDropDown.value + 1
  if index < 0 or index > #servers then
    if GameConfig.LOG_LEVEL <= 4 then
      Log.Error("PanelLogin invalid server index", index, #servers)
    end
    self._serverListHelper:RefreshServerList()
    return
  end
  if self._curSpeedupServer then
    return
  end
  local server = servers[index]
  self._serverListHelper:SetConnectedServer(server)
  local serverName = server.Name
  local ConstProp = require("UniSDK.ConstProp")
  local countryCode = MC.UniSDKHandler:GetPropStr(ConstProp.COUNTRY_CODE)
  mcpcall(function()
    if countryCode == nil or countryCode == "" then
      countryCode = MC.UniSDKHandler:GetPropStr(ConstProp.SDC_LOG_SIM_COUNTRY)
    end
    if countryCode == nil or countryCode == "" then
      local aimInfo = json.decode(MC.UniSDKHandler:GetPropStr(ConstProp.JF_AIM_INFO))
      countryCode = aimInfo.country
    end
  end)
  local serverAddr = server:GetRandomHost(countryCode)
  local info = string.split(serverAddr, ":")
  local ip = ""
  local port = 0
  if #info == 2 then
    ip = info[1]
    port = math.tointeger(info[2]) or 0
  end
  self._curSpeedupServer = {
    serverName = serverName,
    ip = ip,
    port = port,
    server = server
  }
  self._engineVersionError = false
  self._isReachNumMax = false
  if Switch.USE_PHAROS_SPEEDUP and ServerInfo.Instance():NeedSpeedupIp() then
    MC.UniSDKHandler:PharosProbe(ip, tostring(port))
    self:ShowWaitingPanel()
  else
    self:ShowWaitingPanel()
    if Switch.USE_PHAROS_SPEEDUP then
      MC.UniSDKHandler:SetNoPharosSpeedup(ip, port)
    end
    self:OnResponsePharosSpeedup()
  end
end
