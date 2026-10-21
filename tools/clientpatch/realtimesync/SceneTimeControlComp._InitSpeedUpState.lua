function SceneTimeControlComp:_InitSpeedUpState()
  local battle = self._entity:GetBattle()
  local banSpeedUpSwitch, defaultSpeedUp = true, false
  if battle.isRealTimePvP and MC.GameWorld.mmmDuel ~= nil and MC.GameWorld.mmmDuel.manual then
  elseif LogicInterface.CheckSwitch(battle, BattleConst.BattleSwitches.BAN_SPEED_CHANGE) then
  else
    banSpeedUpSwitch = false
    if not LogicInterface.CheckSwitch(battle, BattleConst.BattleSwitches.DEFAULT_NO_SPEEDUP) then
      local globalSpeedUp = MC.UserPrefs:GetBool("battle_speed_up", nil)
      if globalSpeedUp == nil then
        defaultSpeedUp = MC.UserPrefs:GetBool("battle_speed_up" .. SceneTimeControlComp.GetSettingGroup(battle), false)
      else
        defaultSpeedUp = globalSpeedUp
      end
    end
  end
  self._banSpeedUpSwitch = banSpeedUpSwitch
  self._speedUp = defaultSpeedUp
  self:_UpdateSpeedUpState()
end
