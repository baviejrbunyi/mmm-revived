function FriendItem:SetInfo(uid, allData, inManagement, selectDelete, deleteCb)
  self._uid = uid
  self._allData = allData
  self._inManagement = inManagement
  self._selectDelete = selectDelete
  self._deleteCb = deleteCb
  if not self._uobj then
    return
  end
  if not self._duelRightClick then
    self._duelRightClick = true
    pcall(function()
      self:SetForwardCallback(CS.UnityEngine.EventSystems.EventTriggerType.PointerDown, function(_, eventData)
        if eventData.pointerId ~= -2 or not self._uobj or not self._uid or self._inManagement or not self._uobj.activeInHierarchy then
          return
        end
        local hit = eventData.pointerCurrentRaycast.gameObject
        if hit == nil or isnull(hit) or not hit.transform:IsChildOf(self._uobj.transform) then
          return
        end
        UTools.CallNextFrame(function()
          if not self._uobj or not self._uid then
            return
          end
          local opened = MC.UIManager:GetPanel("PanelFriendTip") ~= nil
          local tip = MC.Panel.PanelFriendTip
          tip:Show(self._uid, nil, self._headItem:UObj())
          if opened then
            tip:LayoutPanel()
          end
        end, self)
      end)
    end)
  end
  self._headItem:SetOnHeadClickedCB(bind(self, self._OnHeadClicked))
  self._headItem:SetHead(allData.h)
  self._headItem:SetFrame(allData.hf)
  self._txtServerName.text = MC.UIUtils.GetServerName(allData.logic_hostnum)
  self._txtLv.text = allData.lv or 0
  self._txtName.text = MC.GameWorld.Avatar.BasicComp:GetServerFormatName(allData) or Strings.Get("ACCOUNT_DEACTIVATED")
  self._goDetail:SetActive(allData.sign ~= nil and allData.sign ~= "")
  if allData.sign ~= nil and allData.sign ~= "" then
    self._txtDetail.text = string.trim(allData.sign)
  end
  local online = allData.online or 0
  local offline = allData.offline or 0
  self._goOnline:SetActive(online >= offline)
  if online < offline then
    local offTime = MC.TimeHelper.Time() - offline
    if offTime < Const.DAY_SECONDS * 7 then
      if offTime > Const.DAY_SECONDS then
        self._txtTime.text = Strings.Get("DAY_AGO", {
          math.floor(offTime / Const.DAY_SECONDS)
        })
      elseif offTime > Const.HOUR_SECONDS then
        self._txtTime.text = Strings.Get("HOUR_AGO", {
          math.floor(offTime / Const.HOUR_SECONDS)
        })
      else
        self._txtTime.text = Strings.Get("MINUTE_AGO", {
          math.ceil(offTime / Const.MINUTE_SECONDS)
        })
      end
    else
      self._txtTime.text = Strings.Get("TIME_OFFLINE")
    end
  end
  self._txtTime.gameObject:SetActive(online <= offline)
  self._imgMaskOffLine:SetActive(online < offline)
  self._heroList:RemoveAllItems()
  local supportHeroes = allData.supportHeroes or Const.EmptyTable
  local heroId, data = next(supportHeroes)
  while heroId ~= nil do
    local item = self._heroList:AddItem()
    item:SetHeroInfo(heroId, data)
    item:SetShowCareer(true)
    item:SetShowCamp(true)
    item:SetShowRarity(true)
    item:SetOnClickedCb(bind(self, self._OnHeroClicked))
    item:UpdateAwake()
    heroId, data = next(supportHeroes, heroId)
  end
  self:SetBadgeInfo(allData)
  self:_UpdateGiveBtnState()
  self._goBtn:SetActive(not inManagement)
  self._goNodeDelete:SetActive(inManagement == true)
  self._goDeleteNormal:SetActive(not selectDelete)
  self._goDeleteSelect:SetActive(selectDelete == true)
  self._goBg:SetActive(not inManagement or not selectDelete)
  self._goBgSelect:SetActive(inManagement == true and selectDelete == true)
end
