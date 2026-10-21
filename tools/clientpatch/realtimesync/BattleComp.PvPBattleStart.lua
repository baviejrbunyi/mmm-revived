function BattleComp:PvPBattleStart(randomSeed, playInfo, oppoInfo, environmentId)
  pcall(function()
    local battle = MC.GameWorld.BattleMgrNew:GetCurrentBattle()
    if battle ~= nil then
      battle.realtimeStartAt = CS.UnityEngine.Time.realtimeSinceStartup + 2
    end
  end)
  MC.GameWorld.BattleMgrNew:StartRealTimeBattle(randomSeed, playInfo, oppoInfo, environmentId)
end
