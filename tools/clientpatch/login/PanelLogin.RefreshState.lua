function PanelLogin:RefreshState()
  local showMaintainNotice = false
  local isOverseaMode = PatchUtils:CheckIsGlobalArea()
  local showDebugLogin = self:UseDebugLogin()
  local maintainNoticeState = self._serverListHelper:CheckMaintainNotice()
  if maintainNoticeState == ServerListHelper.MaintainNoticeState.NO_NOTICE then
    showMaintainNotice = false
  elseif maintainNoticeState == ServerListHelper.MaintainNoticeState.DOWNLOADING then
    showMaintainNotice = true
  elseif maintainNoticeState == ServerListHelper.MaintainNoticeState.SHOWING then
    showMaintainNotice = true
  elseif maintainNoticeState == ServerListHelper.MaintainNoticeState.COMPLETED then
    showMaintainNotice = false
  end
  local serverListState = self._serverListHelper:CheckServerListState()
  local serverListDone = serverListState == ServerListHelper.ServerListState.COMPLETED
  serverListDone = serverListDone or self._serverListHelper:GetServerList() ~= nil
  if GameConfig.LOG_LEVEL <= 2 then
    Log.Info("PanelLogin:RefreshState", maintainNoticeState, serverListState)
  end
  self._timeRefreshObj:SetActive(maintainNoticeState == ServerListHelper.MaintainNoticeState.SHOWING)
  self._btnStart:SetActive(not Switch.FORCE_DEBUG_LOGIN and not isOverseaMode and not showMaintainNotice and serverListDone)
  self:FindChildByPath("cont/cn"):SetActive(not isOverseaMode)
  if not isOverseaMode then
    local lbs = {
      lb_1 = "PENEL_ANTI_ADDICTION_TXT",
      lb_2 = "PENEL_ANTI_ADDICTION_TXT_1",
      lb_3 = "PENEL_ANTI_ADDICTION_TXT_2",
      lb_4 = "PENEL_ANTI_ADDICTION_TXT_3"
    }
    local keys = {"lb_1", "lb_2", "lb_3", "lb_4"}
    for i = 1, #keys do
      local k = keys[i]
      self:FindTextByPath("cont/cn/" .. k).text = Strings.Get(lbs[k])
    end
  end
  self._btnLogin:SetActive((Switch.FORCE_DEBUG_LOGIN or isOverseaMode) and not showMaintainNotice and serverListDone)
  self:FindChildByPath("cont/server_choose"):SetActive(GameConfig.IS_DEBUG)
  self:FindChildByPath("cont/language_choose"):SetActive(GameConfig.IS_DEBUG)
  self:FindChildByPath("cont/inp_login"):SetActive(Switch.FORCE_DEBUG_LOGIN and not showMaintainNotice)
  if self.passwordInputField then
    self.passwordInputField.gameObject:SetActive(Switch.FORCE_DEBUG_LOGIN and not showMaintainNotice)
  end
  self:FindChildByPath("cont/btn_switch"):SetActive(false)
  if showMaintainNotice then
    self:UpdateRefreshTime()
    self:SetUpdateEnable(bind(self, self.UpdateRefreshTime), true)
  else
    self:SetUpdateEnable(bind(self, self.UpdateRefreshTime), false)
  end
  self:UpdateServerList()
  self:UpdateVerticalLayout()
end
