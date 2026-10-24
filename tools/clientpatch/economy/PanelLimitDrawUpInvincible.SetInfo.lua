function PanelLimitDrawUpInvincible:SetInfo(data)
  local permanent = data.endTime ~= nil and data.endTime >= 4000000000
  self._timeBg.gameObject:SetActive(not permanent)
  if data.endTime ~= nil and not permanent then
    local timeStr = ""
    local leftTime = math.max(data.endTime - TimeHelper.Time(), 0)
    if leftTime > Const.DAY_SECONDS then
      local day = math.floor(leftTime / Const.DAY_SECONDS)
      local hour = math.floor((leftTime - day * Const.DAY_SECONDS) / Const.HOUR_SECONDS)
      timeStr = string.format("%s %s", Strings.Get("PANEL_DRAW_LEFT_DAY_FMT", {day}), Strings.Get("PANEL_DRAW_LEFT_HOUR_FMT", {hour}))
    elseif leftTime > Const.HOUR_SECONDS then
      local hour = math.floor(leftTime / Const.HOUR_SECONDS)
      timeStr = Strings.Get("PANEL_DRAW_LEFT_HOUR_FMT", {hour})
    else
      local minute = math.floor(leftTime / Const.MINUTE_SECONDS)
      timeStr = Strings.Get("PANEL_DRAW_LEFT_MINUTE_FMT", {minute})
    end
    self._timeText.text = timeStr
  end
  self._titleText.text = data.poolName
  self._titleShadow.text = data.poolName
  self.heroId = data.ShowHero
  if data.extraBackRes then
    self:LoadSprite(data.extraBackRes[1], bind(self, self._OnPicImgLoaded))
  end
  local charData = character_data_new.Get(data.ShowHero)
  if charData then
    local camp = charData.camp
    local campPath = ResourceConst.GetHeroCampImagePath(camp)
    self:LoadSprite(campPath, bind(self, self._OnCampImgLoaded))
    if self._nameText then
      self._nameText.text = charData.name
    end
    if charData.star then
      self._starList:SetItemCount(charData.star)
      local item = self._starList:GetItem(charData.star)
      if item then
        item:HidePoint(true)
      end
    end
  end
  if data.ShowHero2 then
    self._nodeHeadGo:SetActive(true)
    if data.ShowHero2[1] then
      self._headNode1:SetHeadInfo(data.ShowHero2[1])
      self._headNode1:SetClickCallback(bind(self, self._OnHeadClick))
    end
    if data.ShowHero2[2] then
      self._headNode2:SetHeadInfo(data.ShowHero2[2])
      self._headNode2:SetClickCallback(bind(self, self._OnHeadClick))
    end
  end
  local textMeshPro = self._titleText.transform:GetComponent("TextMeshProUGUI")
  if data.nameStyle then
    self:SetTextMeshProColor(textMeshPro, data.nameStyle)
  end
  if data.nameTag then
    local redColor1, greenColor1, blueColor1 = MC.UIUtils.TransformHexToRGB(data.nameTag[1])
    self._nameDi.color = Color(redColor1, greenColor1, blueColor1)
  end
  if data.timeTag then
    local redColor1, greenColor1, blueColor1 = MC.UIUtils.TransformHexToRGB(data.timeTag)
    self._timeBg.color = Color(redColor1, greenColor1, blueColor1)
    self._goUpText.color = Color(redColor1, greenColor1, blueColor1)
  end
  if data.drawNameTexture then
    local fontMatPath = string.format("UI/Fonts/%s.mat", data.drawNameTexture)
    self:LoadExtraAssetWithType(fontMatPath, typeof(CS.UnityEngine.Material), bind(self, self._OnTextMaterialLoaded))
  end
end
