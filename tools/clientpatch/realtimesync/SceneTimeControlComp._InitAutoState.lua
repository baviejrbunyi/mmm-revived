function SceneTimeControlComp:_InitAutoState()
  local battle = self._entity:GetBattle()
  local banSwitchAuto, defaultAuto = true, false
  if battle.isRealTimePvP and MC.GameWorld.mmmDuel ~= nil and MC.GameWorld.mmmDuel.manual then
  elseif battle.clientReplay or LogicInterface.CheckSwitch(battle, BattleConst.BattleSwitches.BAN_AUTO) then
  elseif LogicInterface.CheckSwitch(battle, BattleConst.BattleSwitches.FORCE_AUTO) then
    defaultAuto = true
  else
    banSwitchAuto = false
    if not LogicInterface.CheckSwitch(battle, BattleConst.BattleSwitches.DEFAULT_NO_AUTO) then
      defaultAuto = MC.UserPrefs:GetBool("battle_auto" .. SceneTimeControlComp.GetSettingGroup(battle), false)
    end
  end
  self._banAutoSwitch = banSwitchAuto
  self._isAuto = defaultAuto
  self:_UpdateAutoState()
end
