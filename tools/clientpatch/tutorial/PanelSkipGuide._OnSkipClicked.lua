function PanelSkipGuide:_OnSkipClicked()
  MC.UIManager:OpenPanel("PanelPopUpConfirm", function(panel)
    panel:SetDescribeText("SKIP_ALL_TUTORIALS")
    panel:SetLeftButtonText("CONFIRM_NO")
    panel:SetRightButtonText("CONFIRM_YES")
    panel:SetLeftButton(function()
      MC.UIManager:ClosePanel("PanelPopUpConfirm")
    end)
    panel:SetRightButton(function()
      MC.UIManager:ClosePanel("PanelPopUpConfirm")
      local avatar = MC.GameWorld.Avatar
      local novice = avatar ~= nil and avatar.NoviceComp ~= nil and not avatar.NoviceComp:IsNoviceFinish()
      local current = MC.GuideMgr:GetGuideId()
      if current ~= nil and current ~= 0 then
        MC.GuideMgr:TrySkipGuide(current)
      end
      MC.UIManager:ClosePanel("PanelSkipGuide")
      if avatar == nil then
        return
      end
      avatar:CallServerMethod("skipTutorial")
      if novice then
        avatar.NoviceComp:OnLeaveNovice()
        UTools.DelayCall(MC.GameWorld, 0.5, function()
          MC.GameWorld:EnterLobby()
        end)
      end
    end)
  end)
end
