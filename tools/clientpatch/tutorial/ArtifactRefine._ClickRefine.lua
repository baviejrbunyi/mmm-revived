function ArtifactRefine:_ClickRefine()
  local ret, err = MC.GuideMgr:CheckDoOperation("ART_ENTER_REFINE")
  if not ret then
    MC.UIUtils.ShowNotify(err)
    return
  end
  local isEnough, itemId = self:IsCostEnough()
  local hasAttr = self._subAttrList ~= nil and #self._subAttrList > 0
  if not isEnough or not hasAttr then
    local guideId = MC.GuideMgr:GetGuideId()
    local guide = guideId ~= 0 and config("guide_data").Get(guideId) or nil
    if guide ~= nil and guide.limitType ~= nil and guide.limitType[1] == require("Module.Guide.GuideConst").OperationType.ART_ENTER_REFINE then
      MC.GuideMgr:TrySkipGuide(guideId)
    end
    if not isEnough then
      local itemData = item_data.Get(itemId)
      MC.UIUtils.ShowNotify("ARTIFACT_REFINE_COST_NOT_ENOUGH", {
        itemData.name
      })
    else
      MC.UIUtils.ShowNotify("ARTIFACT_HAVE_NO_SUB_ATTR")
    end
    return
  end
  MC.GuideMgr:OnDoOperation("ART_ENTER_REFINE")
  MC.UIUtils.OpenPanelInSubState("PanelArtifactRefineSub", function(panel)
    panel:SetArtifact(self._artifactId)
  end)
end
