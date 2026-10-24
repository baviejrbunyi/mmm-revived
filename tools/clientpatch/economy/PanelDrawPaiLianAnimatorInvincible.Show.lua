function PanelDrawPaiLianAnimatorInvincible:Show(poolId, cb)
  self.closeCb = cb
  self.poolId = poolId
  local str = string.format("poolId_[%s]_show_anim", poolId)
  MC.UserPrefs:SetBool(str, true)
  MC.GameWorld.Avatar.PopupPackComp:SetShowDrawAnim(true)
  local poolData = draw_pool_data.Get(poolId)
  if poolData then
    self._titleText.text = poolData.poolName
    self._titleShadow.text = poolData.poolName
    local permanent = poolData.endTime ~= nil and poolData.endTime >= 4000000000
    self._timeGo:SetActive(not permanent)
    if poolData.endTime ~= nil and not permanent then
      local timeStr = ""
      local leftTime = math.max(poolData.endTime - TimeHelper.Time(), 0)
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
    if poolData.extraBackRes and poolData.extraBackRes[1] then
      self:LoadSprite(poolData.extraBackRes[1], bind(self, self._OnBackImageLoaded))
    end
    if poolData.ShowHero then
      self.heroId = poolData.ShowHero
      local charData = character_data_new.Get(poolData.ShowHero)
      if charData then
        self._heroName.text = charData.name
        local camp = charData.camp
        local campPath = ResourceConst.GetHeroCampImagePath(camp)
        self:LoadSprite(campPath, bind(self, self._OnCampImgLoaded))
        if charData.star then
          self._starList:SetItemCount(charData.star)
          local item = self._starList:GetItem(charData.star)
          if item then
            item:HidePoint(true)
          end
        end
      end
    end
    if poolData.heroShowTimeline then
      self:_TryPlayTimeline(poolData.heroShowTimeline)
    end
    if poolData.bgm then
      local stage = MC.StageManager:GetActiveStage()
      safecall(stage.StopBgm, stage)
      local musicData = MC.AudioManager:GetEventMusic(poolData.bgm)
      self._playingMusicId = MC.AudioManager:PlayEvent(musicData.eventName)
    end
    local textMeshPro = self._titleText.transform:GetComponent("TextMeshProUGUI")
    if poolData.nameStyle then
      self:SetTextMeshProColor(textMeshPro, poolData.nameStyle)
    end
    if poolData.nameTag then
      local redColor1, greenColor1, blueColor1 = MC.UIUtils.TransformHexToRGB(poolData.nameTag[1])
      self._nameDi.color = Color(redColor1, greenColor1, blueColor1)
    end
    if poolData.timeTag then
      local redColor1, greenColor1, blueColor1 = MC.UIUtils.TransformHexToRGB(poolData.timeTag)
      self._timeBg.color = Color(redColor1, greenColor1, blueColor1)
    end
    if poolData.drawNameTexture then
      local fontMatPath = string.format("UI/Fonts/%s.mat", poolData.drawNameTexture)
      self:LoadExtraAssetWithType(fontMatPath, typeof(CS.UnityEngine.Material), bind(self, self._OnTextMaterialLoaded))
    end
  end
end
