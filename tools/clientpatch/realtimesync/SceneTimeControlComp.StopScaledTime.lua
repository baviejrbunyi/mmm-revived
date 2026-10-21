function SceneTimeControlComp:StopScaledTime(controller, onStop)
  if onStop == nil then
    return
  end
  local ok, realtime = pcall(function()
    local stage = MC.StageManager:GetActiveStage()
    return stage ~= nil and stage.battle ~= nil and stage.battle.isRealTimePvP == true
  end)
  if ok and realtime then
    onStop()
    return
  end
  self._entity.SceneTaskExecComp:StartTask(controller, function()
    self:SetTimeScale(0)
    onStop()
  end)
end
