function Battle:CheckSwitch(switch)
  if self.isRealTimePvP and MC.GameWorld.mmmDuel ~= nil and MC.GameWorld.mmmDuel.manual then
    if switch == BattleConst.BattleSwitches.SELECT_TARGET_NO_PAUSE then
      return 1
    end
    if switch == BattleConst.BattleSwitches.FORCE_AUTO then
      return nil
    end
  end
  if not self.switches then
    self.switches = self:GetConfig().switches or {}
  end
  return self.switches[switch]
end
