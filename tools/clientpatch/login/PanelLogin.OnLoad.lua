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
  pcall(function()
    if rawget(_G, "MMMRepl") ~= nil then
      return
    end
    local args = CS.System.Environment.GetCommandLineArgs()
    local file
    for i = 0, args.Length - 1 do
      if string.find(args[i], "-mmmCommand=", 1, true) == 1 then
        file = string.sub(args[i], 13)
      end
    end
    if file == nil then
      return
    end
    local repl = {done = 0}
    rawset(_G, "MMMRepl", repl)
    local Vector2 = CS.UnityEngine.Vector2
    local function emit(text)
      local handle = io.open(file .. ".out", "a")
      if handle then
        handle:write(text .. "\n")
        handle:close()
      end
    end
    local function split(text, separator)
      local parts = {}
      local start = 1
      while true do
        local stop = string.find(text, separator, start, true)
        if stop == nil then
          parts[#parts + 1] = string.sub(text, start)
          return parts
        end
        parts[#parts + 1] = string.sub(text, start, stop - 1)
        start = stop + 1
      end
    end
    local function resolve(path)
      local node = _G
      local parent, key
      local start = 1
      local method = false
      while true do
        local stop = string.find(path, "[%.:]", start)
        local part = stop and string.sub(path, start, stop - 1) or string.sub(path, start)
        parent = node
        key = tonumber(part) or part
        if node == nil then
          return nil, parent, key, method
        end
        node = node[key]
        if stop == nil then
          return node, parent, key, method
        end
        method = string.sub(path, stop, stop) == ":"
        start = stop + 1
      end
    end
    local dump
    function dump(value, depth)
      local kind = type(value)
      if kind == "string" then
        return string.format("%q", value)
      end
      if kind ~= "table" then
        return tostring(value)
      end
      if depth <= 0 then
        return "{...}"
      end
      local items = {}
      local key, item = next(value)
      local count = 0
      while key ~= nil and count < 600 do
        items[#items + 1] = tostring(key) .. "=" .. dump(item, depth - 1)
        count = count + 1
        key, item = next(value, key)
      end
      if key ~= nil then
        items[#items + 1] = "..."
      end
      return "{" .. table.concat(items, ", ") .. "}"
    end
    local function literal(text)
      local prefix, rest = string.sub(text, 1, 2), string.sub(text, 3)
      if prefix == "s:" then
        return rest
      elseif prefix == "n:" then
        return tonumber(rest)
      elseif prefix == "b:" then
        return rest == "true"
      elseif prefix == "p:" then
        return (resolve(rest))
      elseif prefix == "t:" then
        local result = {}
        if rest ~= "" then
          local entries = split(rest, ",")
          for index = 1, #entries do
            local entry = split(entries[index], "=")
            local key = tonumber(entry[1]) or entry[1]
            local value = entry[2]
            result[key] = tonumber(value) or value
          end
        end
        return result
      elseif text == "nil" then
        return nil
      end
      return tonumber(text) or text
    end
    local function screen()
      return CS.UnityEngine.Screen.width, CS.UnityEngine.Screen.height
    end
    local function pathOf(transform)
      local names = {}
      local node = transform
      while node ~= nil and not isnull(node) do
        table.insert(names, 1, node.name)
        node = node.parent
      end
      return table.concat(names, "/")
    end
    local function camera()
      return MC.UIManager:Root():GetComponent(typeof(CS.UnityEngine.Canvas)).worldCamera
    end
    local function positionOf(transform)
      local view = camera()
      local canvas = transform:GetComponentInParent(typeof(CS.UnityEngine.Canvas))
      if canvas ~= nil and not isnull(canvas) then
        local root = canvas.rootCanvas
        if root.renderMode == CS.UnityEngine.RenderMode.ScreenSpaceOverlay then
          view = nil
        elseif root.worldCamera ~= nil and not isnull(root.worldCamera) then
          view = root.worldCamera
        end
      end
      local point = CS.UnityEngine.RectTransformUtility.WorldToScreenPoint(view, transform.position)
      local width, height = screen()
      return point.x, point.y, point.x / width, 1 - point.y / height
    end
    local function guideTarget()
      local id = MC.GuideMgr:GetGuideId()
      if id == nil or id == 0 then
        return nil, "no guide"
      end
      local data = config("guide_data").Get(id)
      local helper = require("Module.Guide.PosHelper")
      local position = data.maskPosType and helper.GetMaskPosition(data) or nil
      if position == nil and data.aniPosType then
        position = helper.GetAnimationPosition(data)
      end
      if position == nil then
        return nil, "guide " .. id .. " has no target"
      end
      local point = camera():WorldToScreenPoint(CS.UnityEngine.Vector3(position.x, position.y, position.z or 0))
      local target = nil
      if data.aniTarPosType then
        local raw = helper.GetAnimationTargetPosition(data)
        if raw ~= nil then
          target = camera():WorldToScreenPoint(CS.UnityEngine.Vector3(raw.x, raw.y, raw.z or 0))
        end
      end
      return point, id, data, target
    end
    local function raycast(x, y)
      local system = CS.UnityEngine.EventSystems.EventSystem.current
      local pointer = CS.UnityEngine.EventSystems.PointerEventData(system)
      pointer.position = Vector2(x, y)
      local results = CS.System.Collections.Generic.List(CS.UnityEngine.EventSystems.RaycastResult)()
      system:RaycastAll(pointer, results)
      return pointer, results
    end
    local function invoke(target, name, pointer)
      local components = target:GetComponents(typeof(CS.UnityEngine.MonoBehaviour))
      local handled = false
      for index = 0, components.Length - 1 do
        local component = components[index]
        if component ~= nil and not isnull(component) and component.enabled then
          local ok = pcall(function()
            local method = component[name]
            if method ~= nil then
              method(component, pointer)
              handled = true
            end
          end)
          if not ok then
            handled = handled or false
          end
        end
      end
      return handled
    end
    local function handlerOf(start, name)
      local node = start.transform
      while node ~= nil and not isnull(node) do
        local components = node.gameObject:GetComponents(typeof(CS.UnityEngine.MonoBehaviour))
        for index = 0, components.Length - 1 do
          local component = components[index]
          local found = false
          pcall(function()
            found = component ~= nil and component.enabled and component[name] ~= nil
          end)
          if found then
            return node.gameObject
          end
        end
        node = node.parent
      end
      return nil
    end
    local function forward(kind, pointer)
      pcall(function()
        MC.EventForwardMgr:_OnEvent(CS.UnityEngine.EventSystems.EventTriggerType[kind], pointer)
      end)
    end
    local function clickAt(x, y)
      local pointer, results = raycast(x, y)
      forward("PointerDown", pointer)
      if results.Count == 0 then
        return "nothing hit"
      end
      local first = results[0]
      pointer.pointerCurrentRaycast = first
      pointer.pointerPressRaycast = first
      local hit = first.gameObject
      local down = handlerOf(hit, "OnPointerDown")
      if down ~= nil then
        invoke(down, "OnPointerDown", pointer)
      end
      local click = handlerOf(hit, "OnPointerClick")
      pointer.pointerPress = click or down
      if down ~= nil then
        invoke(down, "OnPointerUp", pointer)
      end
      if click ~= nil then
        invoke(click, "OnPointerClick", pointer)
      end
      forward("PointerUp", pointer)
      forward("PointerClick", pointer)
      return "hit " .. pathOf(hit.transform) .. " click " .. tostring(click and pathOf(click.transform))
    end
    local function holdAt(x, y, seconds)
      local pointer, results = raycast(x, y)
      forward("PointerDown", pointer)
      if results.Count == 0 then
        return "nothing hit"
      end
      local first = results[0]
      pointer.pointerCurrentRaycast = first
      pointer.pointerPressRaycast = first
      local hit = first.gameObject
      local down = handlerOf(hit, "OnPointerDown")
      if down ~= nil then
        invoke(down, "OnPointerDown", pointer)
      end
      pointer.pointerPress = down
      UTools.DelayCall(nil, seconds, function()
        if down ~= nil then
          invoke(down, "OnPointerUp", pointer)
        end
        forward("PointerUp", pointer)
      end, true)
      return "holding " .. pathOf(hit.transform)
    end
    local function dragFrom(x1, y1, x2, y2)
      local pointer, results = raycast(x1, y1)
      forward("PointerDown", pointer)
      if results.Count == 0 then
        return "nothing hit"
      end
      local first = results[0]
      pointer.pointerCurrentRaycast = first
      pointer.pointerPressRaycast = first
      local hit = first.gameObject
      local down = handlerOf(hit, "OnPointerDown")
      local drag = handlerOf(hit, "OnBeginDrag") or down
      if down ~= nil then
        invoke(down, "OnPointerDown", pointer)
      end
      pointer.pointerPress = down
      pointer.pointerDrag = drag
      pointer.dragging = true
      if drag ~= nil then
        invoke(drag, "OnBeginDrag", pointer)
      end
      local steps = 8
      local index = 1
      while index <= steps do
        pointer.position = Vector2(x1 + (x2 - x1) * index / steps, y1 + (y2 - y1) * index / steps)
        local _, under = raycast(pointer.position.x, pointer.position.y)
        if under.Count > 0 then
          pointer.pointerCurrentRaycast = under[0]
        end
        if drag ~= nil then
          invoke(drag, "OnDrag", pointer)
        end
        index = index + 1
      end
      if drag ~= nil then
        invoke(drag, "OnEndDrag", pointer)
      end
      if down ~= nil then
        invoke(down, "OnPointerUp", pointer)
      end
      forward("PointerUp", pointer)
      return "dragged " .. pathOf(hit.transform) .. " via " .. tostring(drag and pathOf(drag.transform))
    end
    local function find(path)
      local object = CS.UnityEngine.GameObject.Find(path)
      if object == nil or isnull(object) then
        return nil
      end
      return object
    end
    local function listTexts(filter)
      local lines = {}
      local kinds = {typeof(CS.UnityEngine.UI.Text)}
      pcall(function()
        kinds[#kinds + 1] = typeof(CS.TMPro.TMP_Text)
      end)
      for kindIndex = 1, #kinds do
        local found = CS.UnityEngine.Object.FindObjectsOfType(kinds[kindIndex])
        for index = 0, found.Length - 1 do
          local item = found[index]
          local text = item.text or ""
          if item.gameObject.activeInHierarchy and text ~= "" and (filter == "" or string.find(string.lower(text), string.lower(filter), 1, true)) then
            local _, _, nx, ny = positionOf(item.transform)
            lines[#lines + 1] = string.format("%.3f,%.3f %s :: %s", nx, ny, string.gsub(text, "\n", " "), pathOf(item.transform))
          end
        end
      end
      return table.concat(lines, "\n")
    end
    local function listButtons(filter)
      local lines = {}
      local kinds = {typeof(CS.UnityEngine.UI.Button), typeof(CS.UnityEngine.EventSystems.EventTrigger), typeof(CS.UnityEngine.UI.Toggle)}
      for kindIndex = 1, #kinds do
        local found = CS.UnityEngine.Object.FindObjectsOfType(kinds[kindIndex])
        for index = 0, found.Length - 1 do
          local item = found[index]
          local path = pathOf(item.transform)
          if item.gameObject.activeInHierarchy and (filter == "" or string.find(string.lower(path), string.lower(filter), 1, true)) then
            local _, _, nx, ny = positionOf(item.transform)
            lines[#lines + 1] = string.format("%.3f,%.3f %s", nx, ny, path)
          end
        end
      end
      return table.concat(lines, "\n")
    end
    local function run(line)
      local parts = split(line, "|")
      local verb = parts[1]
      if verb == "get" then
        return dump((resolve(parts[2])), tonumber(parts[3]) or 2)
      elseif verb == "set" then
        local _, parent, key = resolve(parts[2])
        parent[key] = literal(parts[3])
        return "ok"
      elseif verb == "call" then
        local fn, parent, _, method = resolve(parts[2])
        local values = {}
        local count = 0
        if method then
          count = 1
          values[1] = parent
        end
        for index = 3, #parts do
          count = count + 1
          values[count] = literal(parts[index])
        end
        local results = table.pack(fn(table.unpack(values, 1, count)))
        local rendered = {}
        for index = 1, results.n do
          rendered[index] = dump(results[index], tonumber(parts[#parts + 1]) or 2)
        end
        return table.concat(rendered, " ; ")
      elseif verb == "rpc" then
        local values = {}
        for index = 3, #parts do
          values[index - 2] = literal(parts[index])
        end
        MC.GameWorld.Avatar:CallServerMethod(parts[2], table.unpack(values, 1, #parts - 2))
        return "sent"
      elseif verb == "gm" then
        MC.GameWorld.Avatar:CallServerMethod("gmcmd", parts[2])
        return "sent"
      elseif verb == "click" then
        local object = find(parts[2])
        if object == nil then
          return "not found"
        end
        local x, y = positionOf(object.transform)
        return clickAt(x, y)
      elseif verb == "clickxy" then
        local width, height = screen()
        return clickAt(tonumber(parts[2]) * width, (1 - tonumber(parts[3])) * height)
      elseif verb == "hits" then
        local width, height = screen()
        local _, results = raycast(tonumber(parts[2]) * width, (1 - tonumber(parts[3])) * height)
        local lines = {}
        for index = 0, math.min(results.Count, 6) - 1 do
          lines[#lines + 1] = pathOf(results[index].gameObject.transform)
        end
        return table.concat(lines, "\n")
      elseif verb == "press" then
        local object = find(parts[2])
        if object == nil then
          return "not found"
        end
        local button = object:GetComponent(typeof(CS.UnityEngine.UI.Button))
        if button ~= nil and not isnull(button) then
          button.onClick:Invoke()
          return "invoked button"
        end
        local width, height = screen()
        local x, y = positionOf(object.transform)
        return clickAt(x, y)
      elseif verb == "guide" then
        local point, id, data = guideTarget()
        if point == nil then
          return id
        end
        local width, height = screen()
        local _, _, _, target = guideTarget()
        local aim = target and string.format(" target=%.3f,%.3f", target.x / width, 1 - target.y / height) or ""
        return string.format("guide=%s type=%s limit=%s pos=%.3f,%.3f%s text=%s", tostring(id), tostring(data.type), dump(data.limitType, 1), point.x / width, 1 - point.y / height, aim, tostring(data.textContent))
      elseif verb == "dragguide" then
        local point, id, data, target = guideTarget()
        if point == nil then
          return id
        end
        if target == nil then
          return "guide " .. id .. " has no drag target"
        end
        return "guide " .. id .. " " .. dragFrom(point.x, point.y, target.x, target.y)
      elseif verb == "hold" then
        local width, height = screen()
        return holdAt(tonumber(parts[2]) * width, (1 - tonumber(parts[3])) * height, tonumber(parts[4]) or 1.5)
      elseif verb == "drag" then
        local width, height = screen()
        return dragFrom(tonumber(parts[2]) * width, (1 - tonumber(parts[3])) * height, tonumber(parts[4]) * width, (1 - tonumber(parts[5])) * height)
      elseif verb == "tapguide" then
        local point, id = guideTarget()
        if point == nil then
          return id
        end
        return "guide " .. id .. " " .. clickAt(point.x, point.y)
      elseif verb == "rcall" then
        local module = require(parts[2])
        local fn = module[parts[3]]
        local values = {}
        for index = 4, #parts do
          values[index - 3] = literal(parts[index])
        end
        local results = table.pack(fn(table.unpack(values, 1, #parts - 3)))
        local rendered = {}
        for index = 1, results.n do
          rendered[index] = dump(results[index], 2)
        end
        return table.concat(rendered, " ; ")
      elseif verb == "lua" then
        local handle = io.open(parts[2], "rb")
        if handle == nil then
          return "missing " .. tostring(parts[2])
        end
        local bytes = handle:read("a")
        handle:close()
        local chunk, problem = load(bytes, "=eval", "b")
        if chunk == nil then
          return "compile " .. tostring(problem)
        end
        return dump(chunk(), 3)
      elseif verb == "texts" then
        return listTexts(parts[2] or "")
      elseif verb == "buttons" then
        return listButtons(parts[2] or "")
      elseif verb == "state" then
        local avatar = MC.GameWorld.Avatar
        local stage = MC.StageManager:GetActiveStage()
        local basic = avatar and avatar.BasicComp
        return string.format("stage=%s guide=%s lv=%s exp=%s novice=%s energy=%s", tostring(stage and stage:GetStageType()), tostring(MC.GuideMgr:GetGuideId()), tostring(basic and basic.lv), tostring(basic and basic.exp), tostring(avatar and avatar.NoviceComp and avatar.NoviceComp.noviceId), tostring(avatar and avatar.ItemComp and avatar.ItemComp:GetItemCount(6)))
      end
      return "unknown verb " .. tostring(verb)
    end
    local function poll()
      pcall(function()
        local handle = io.open(file, "r")
        if handle == nil then
          return
        end
        local content = handle:read("a")
        handle:close()
        local lines = split(content, "\n")
        for index = 1, #lines do
          local line = string.gsub(lines[index], "\r", "")
          local space = string.find(line, " ", 1, true)
          local id = space and tonumber(string.sub(line, 1, space - 1))
          if id ~= nil and id > repl.done then
            repl.done = id
            local ok, result = pcall(run, string.sub(line, space + 1))
            emit(string.format("#%d %s %s", id, ok and "ok" or "error", tostring(result)))
          end
        end
      end)
      UTools.DelayCall(nil, 0.3, poll, true)
    end
    pcall(function()
      local handle = io.open(file, "r")
      if handle ~= nil then
        local lines = split(handle:read("a"), "\n")
        handle:close()
        for index = 1, #lines do
          local space = string.find(lines[index], " ", 1, true)
          local id = space and tonumber(string.sub(lines[index], 1, space - 1))
          if id ~= nil and id > repl.done then
            repl.done = id
          end
        end
      end
    end)
    emit("#0 ready " .. tostring(repl.done))
    UTools.DelayCall(nil, 0.3, poll, true)
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
