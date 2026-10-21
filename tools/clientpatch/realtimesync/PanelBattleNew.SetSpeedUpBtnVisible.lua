function PanelBattleNew:SetSpeedUpBtnVisible(visible)
  local duel = MC.GameWorld.mmmDuel
  if self._battle ~= nil and self._battle.isRealTimePvP and duel ~= nil and duel.manual then
    visible = false
    self._countdownTargetTime = nil
    if self._goTime ~= nil then
      self._goTime:SetActive(false)
    end
  end
  self._btnSpeed:SetVisible(visible)
end
