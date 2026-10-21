function BattleNewStage:ShowGuide()
  if self.battleModeData and self._guideController ~= nil then
    self._guideController:Destroy()
    self._guideController = nil
  end
  local guide = self.battleModeData.guide
  if CustomGuideConfig.BATTLE[self.battleModeId] ~= nil or guide and 0 < #guide then
    local passBattle = MC.UserPrefs:GetBool("battle_guide_" .. tostring(self.battleModeId), false)
    local onceBattle = MC.UserPrefs:GetBool("battle_guide_once_" .. tostring(self.battleModeId), false)
    if self.battle:CheckSwitch(BattleConst.BattleSwitches.ONE_TIME_GUIDE) and onceBattle then
      return
    elseif self.battle:CheckSwitch(BattleConst.BattleSwitches.SKIP_GUIDE) and passBattle then
      if self:isFromMaterial() then
        return
      end
      self:GetSceneEntity().SceneBulletTimeComp:SetBulletTime(0, false, false, self)
      MC.UIManager:OpenPanel("PanelPopUpConfirm", function(panel)
        panel:SetDescribeText("SKIP_MEMENTO_GUIDE")
        panel:SetLeftButtonText("CONFIRM_NO")
        panel:SetRightButtonText("CONFIRM_YES")
        panel:SetDisableBack(true)
        panel:SetLeftButton(function()
          MC.UIManager:ClosePanel("PanelPopUpConfirm")
          self:AddGuideController()
          self:GetSceneEntity().SceneBulletTimeComp:CancelBulletTime(self)
        end)
        panel:SetRightButton(function()
          MC.UIManager:ClosePanel("PanelPopUpConfirm")
          self:GetSceneEntity().SceneBulletTimeComp:CancelBulletTime(self)
          local avatar = MC.GameWorld.Avatar
          if avatar ~= nil and avatar.NoviceComp ~= nil and not avatar.NoviceComp:IsNoviceFinish() then
            avatar:CallServerMethod("skipTutorial")
            avatar.NoviceComp:OnLeaveNovice()
            UTools.DelayCall(MC.GameWorld, 0.5, function()
              MC.GameWorld:EnterLobby()
            end)
          end
        end)
      end)
    else
      self:AddGuideController()
    end
  end
end
