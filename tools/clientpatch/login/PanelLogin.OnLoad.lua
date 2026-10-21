function PanelLogin:OnLoad()
  super(PanelLogin):OnLoad()
  self:SetOnPanelLayoutComplete(bind(self, self.UpdateVerticalLayout))
  MC.UniSDKHandler.DRPF:ReachLoginPanel()
  self:FilterLowDevice()
  self.imgLogo = self:FindImageByPath("cont/img_logo")
  MC.UIUtils.LoadSpriteToImage(self, self.imgLogo, ResourceConst.GetLogoImgPath())
  self.userNameInputField = self:FindChildCompByPath("cont/inp_login", typeof(InputField))
  local userObject = self.userNameInputField.gameObject
  local passwordObject = CS.UnityEngine.Object.Instantiate(userObject, userObject.transform.parent, false)
  passwordObject.name = "inp_password"
  self.passwordInputField = passwordObject:GetComponent(typeof(InputField))
  self.passwordInputField.contentType = InputField.ContentType.Password
  self.passwordInputField.lineType = InputField.LineType.SingleLine
  self.passwordInputField.characterLimit = 256
  self.passwordInputField.text = ""
  self.passwordInputField.placeholder:GetComponent(typeof(CS.UnityEngine.UI.Text)).text = "Password"
  self.userNameInputField.characterLimit = 64
  self._imgVoiceIcon = self:FindImageByPath("cont/btn_enter/btn_voice/pressed/cont/img_icon")
  self:AddButtonListener("cont/btn_enter/btn_voice", bind(self, self.OnClickVoice))
  self:AddButtonListener("cont/btn_enter/btn_notice", bind(self, self.OnNoticeClicked))
  self:AddButtonListener("cont/btn_enter/btn_fix", bind(self, self.OnClickFix))
  local fixButton = self:FindChildByPath("cont/btn_enter/btn_fix")
  if fixButton then
    fixButton:SetActive(false)
  end
  self:AddButtonListener("cont/btn_enter/btn_agreement", bind(self, self.OnClickAgreement))
  self:AddButtonListener("cont/btn_enter/btn_cmp", bind(self, self.OnClickCMP))
  self:AddButtonListener("cont/btn_enter/btn_role", bind(self, self.OnClickRole))
  self:AddButtonListener("cont/btn_enter/btn_quit", bind(self, self.OnClickQuit))
  self:AddButtonListener("cont/btn_enter/btn_service", bind(self, self.OnClickService))
  self:AddButtonListener("cont/btn_enter/btn_language", bind(self, self.OnClickLanguage))
  if not MC.UniSDKHandler:CheckChannelSwitch(Const.ChannelSwitches.Notice) then
    self:FindChildByPath("cont/btn_enter/btn_notice"):SetActive(false)
  end
  self:FindChildByPath("cont/btn_enter/btn_service"):SetActive(true)
  if not MC.UniSDKHandler:CheckChannelSwitch(Const.ChannelSwitches.CustomerService) or not MC.UniSDKHandler.GMBridge:CanOpenGMPage() then
    self:FindChildByPath("cont/btn_enter/btn_service"):SetActive(false)
  end
  if not MC.UniSDKHandler:GetSdkU3d():hasFeature("FEATURE_HAS_MANAGER") then
    self:FindChildByPath("cont/btn_enter/btn_role"):SetActive(false)
  end
  local languageVisible = #MC.I18NHelper.ALL_VALID_LANGUAGE > 1
  self:FindChildByPath("cont/btn_enter/btn_language"):SetActive(languageVisible)
  self._goCmp = self:FindChildByPath("cont/btn_enter/btn_cmp")
  self._goCmp:SetActive(MC.UniSDKHandler:GetPropStr("NT_CMP_NOT_IN_EEA") == "0")
  self:AddButtonListener("cont/btn_login", bind(self, self.OnClickLogin))
  self._btnLogin = self:FindChildByPath("cont/btn_login")
  self:AddButtonListener("cont/btn_switch", bind(self, self._SwitchForceDebugLogin))
  local switchButton = self:FindChildByPath("cont/btn_switch")
  if switchButton then
    switchButton:SetActive(false)
  end
  self._btnCadpa16 = self:FindChildByPath("cont/cn/btn_cadpa16")
  self:AddButtonListener("cont/cn/btn_cadpa16", bind(self, self.OnClickCadpa16))
  self:AddButtonListener("cont/cn/btn_start", bind(self, self.OnClickLogin))
  self._btnStart = self:FindChildByPath("cont/cn/btn_start")
  self:FindChildByPath("cont/server"):SetActive(false)
  self._timeRefreshObj = self:FindChildByPath("cont/time_refresh")
  self._timeRefreshContent = self:FindTextByPath("cont/time_refresh/content")
  self._timeRefreshTime = self:FindTextByPath("cont/time_refresh/time")
  self:AddButtonListener("cont/time_refresh/btn_refresh", bind(self, self.OnClickRefresh))
  local txtRefresh = self:FindTextByPath("cont/time_refresh/btn_refresh/cont/lb_login")
  if txtRefresh then
    txtRefresh.text = Strings.Get("SEASON_COMPETITION_BUTTON_REFRESH")
  end
  local accountPlaceHold = self:FindChildCompByPath("cont/inp_login/lb_account_default", typeof(CS.UnityEngine.UI.Text))
  if accountPlaceHold then
    accountPlaceHold.text = Strings.Get("LOGIN_INPUT_ACCOUNT_TIPS")
  end
  if self:UseDebugLogin() then
    self.userNameInputField.text = PlayerPrefs.GetString("LoginUserName", "")
    if PlayerPrefs.HasKey("CUSTOM_FORCE_DEBUG_LOGIN") then
      Switch.FORCE_DEBUG_LOGIN = PlayerPrefs.GetInt("CUSTOM_FORCE_DEBUG_LOGIN") == 1
    end
  else
    Switch.FORCE_DEBUG_LOGIN = false
  end
  Switch.FORCE_DEBUG_LOGIN = true
  local lbVersion = self:FindChildCompByPath("cont/lb_version", typeof(CS.UnityEngine.UI.Text))
  if lbVersion then
    lbVersion.text = string.format([[
app version: %s    patch version: %s
branch: %s]], PatchUtils:GetAppVersion(), PatchUtils:GetClientVersion(), PatchUtils:GetBranch())
  end
  self.languageChoose = self:FindChildCompByPath("cont/language_choose", typeof(Dropdown))
  self.languageChoose:ClearOptions()
  for idx = 1, #I18NHelper.ALL_LANGUAGE do
    local language = I18NHelper.ALL_LANGUAGE[idx]
    local optionData = Dropdown.OptionData()
    optionData.text = I18NHelper.LANGUAGE_TEXT[language] or language
    self.languageChoose.options:Add(optionData)
    if language == I18NHelper.CUR_LANGUAGE then
      self.languageChoose.value = idx - 1
    end
  end
  self.languageChoose.onValueChanged:AddListener(function(idx)
    local configItem = require("Helper.GameSettingHelper").GetConfigItem("language")
    safecall(configItem.configInfo.language_check_func, I18NHelper.ALL_LANGUAGE[idx + 1], bind(self, self.OnLanguageChanged))
  end)
  I18NHelper.AddLanguageChangeListener(bind(self, self.OnLanguageChanged))
  self._serverDropDown = self:FindChildCompByPath("cont/server_choose", typeof(Dropdown))
  self._serverDropDown:ClearOptions()
  MC.UniSDKHandler:EnableOverSeaChild()
  local lbVersionText = self:FindChildCompByPath("cont/lb_version", "TMP_Text")
  if lbVersionText then
    safexpcall(function()
      local constant_data = config("constant_data")
      local clientVersion = PatchUtils:GetClientVersion()
      local clientVersion3 = 0
      if clientVersion then
        local splitted = string.split(clientVersion, ".")
        clientVersion3 = splitted[3]
      end
      local desc = string.format("%s.%s.%s", constant_data.game_version_first, constant_data.game_version_second, clientVersion3)
      lbVersionText.text = Strings.Get("GAME_VERSION_DESC", {desc})
    end)
  end
  self._serverListHelper = ServerListHelper.Instance()
  local patchStatus = require("Patch.Status")
  patchStatus:SetPercent(120)
  patchStatus:ClosePatchUI()
  MC.UIManager:ClosePanel("PanelDownload")
  MC.GameWorld.InputKeyMgr:RegisterClick(CS.UnityEngine.KeyCode.Return, bind(self, self.OnReturn))
  MC.GameWorld.InputKeyMgr:RegisterClick(CS.UnityEngine.KeyCode.KeypadEnter, bind(self, self.OnReturn))
  MC.GameWorld.TopLayer:ClearUIDWaterMark()
  local downloadBtn = self:AddSimpleChildUI("UI.Common.Download.DownloadBtn", "cont/btn_enter/download_btn")
  downloadBtn:SetConfig(false)
  MC.DownloadMgr:TryInitDLInfo()
  MC.EventUtil.AddHandlers(self, {
    [EventDef.ON_REQUEST_SERVER_LIST] = bind(self, self.RefreshState),
    [EventDef.ON_REQUEST_MAINTAIN_NOTICE] = bind(self, self.RefreshState),
    [EventDef.UI_CANVAS_LAYOUT_CHANGED] = bind(self, self.DelayUpdateVerticalLayout),
    [EventDef.ON_PHAROS_SPEEDUP] = bind(self, self.OnResponsePharosSpeedup),
    [EventDef.ENGINE_VERSION_ERROR] = bind(self, self.OnEngineVersionError),
    [EventDef.PLAYER_NUM_REACH_MAX] = bind(self, self.OnReachNumMax),
    [EventDef.UNISDK_OPEN_LOGIN] = bind(self, self.OnUnisdkLogin)
  })
  GameConfig.ServersInfo = {}
  ServerListHelper.Instance():Start()
  if GameConfig.LOG_LEVEL <= 2 then
    Log.Info("[PanelLogin] ServerListHelper Start")
  end
  self:RefreshState()
  MC.UniSDKHandler:CacheIpv6SpeedTestServer(Switch.USE_PHAROS_SPEEDUP)
  mcpcall(self.ReplaceBackground, self)
  local function watchScreen()
    if self.imgLogo == nil or isnull(self.imgLogo) then
      return
    end
    local width, height = CS.UnityEngine.Screen.width, CS.UnityEngine.Screen.height
    if width ~= self.mmmScreenWidth or height ~= self.mmmScreenHeight then
      self.mmmScreenWidth, self.mmmScreenHeight = width, height
      self:UpdateVerticalLayout()
      UTools.DelayCall(self, 0.1, function()
        if self.imgLogo ~= nil and not isnull(self.imgLogo) then
          self:UpdateVerticalLayout()
        end
      end)
    end
    UTools.DelayCall(self, 0.25, watchScreen)
  end
  UTools.DelayCall(self, 0.25, watchScreen)
  pcall(function()
    local args = CS.System.Environment.GetCommandLineArgs()
    local spec
    for i = 0, args.Length - 1 do
      if string.find(args[i], "-mmmAutotest=", 1, true) == 1 then
        spec = string.sub(args[i], 14)
      end
    end
    if spec == nil then
      return
    end
    local fields = {}
    local start = 1
    while true do
      local stop = string.find(spec, ":", start, true)
      if stop == nil then
        fields[#fields + 1] = string.sub(spec, start)
        break
      end
      fields[#fields + 1] = string.sub(spec, start, stop - 1)
      start = stop + 1
    end
    self.userNameInputField.text = fields[1] or ""
    self.passwordInputField.text = fields[2] or ""
    print("[mmmtest] login", fields[1])
    UTools.DelayCall(self, 3, function()
      self:OnClickLogin()
    end)
  end)
  if self:CheckShowCG() then
    return
  end
  self:StartLogin()
  MC.GameWorld.NoticeMgr:SetAfterRequestCb(function(content)
    MC.Panel.PanelNotice:Show(content, MC.UIConst.UINoticeShowType.Login)
  end)
  MC.GameWorld.NoticeMgr:RequestNotice()
end
