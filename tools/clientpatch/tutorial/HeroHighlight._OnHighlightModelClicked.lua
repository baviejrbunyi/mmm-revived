function HeroHighlight:_OnHighlightModelClicked()
  local ret, err = MC.GuideMgr:CheckDoOperation("BM_ENTER_EQUIP_HIGHLIGHT")
  if not ret then
    MC.UIUtils.ShowNotify(err)
    return
  end
  local guideId = MC.GuideMgr:GetGuideId()
  local guide = guideId ~= 0 and config("guide_data").Get(guideId) or nil
  local list = self:GetHighlightListData()
  if guide ~= nil and guide.limitType ~= nil and guide.limitType[1] == 609 and (list == nil or #list == 0) then
    MC.GuideMgr:TrySkipGuide(guideId)
  else
    MC.GuideMgr:OnDoOperation("BM_ENTER_EQUIP_HIGHLIGHT")
  end
  self:_TryEnterEquip()
end
