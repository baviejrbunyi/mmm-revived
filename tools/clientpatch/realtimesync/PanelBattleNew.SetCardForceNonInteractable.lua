function PanelBattleNew:SetCardForceNonInteractable()
  local duel = MC.GameWorld.mmmDuel
  if duel ~= nil and duel.manual and self._battle ~= nil and self._battle.isRealTimePvP then
    self._forceNonInteractable = false
    self:SetCardInteractable(true)
    return
  end
  self:SetCardInteractable(false)
  self._forceNonInteractable = true
end
