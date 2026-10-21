function ResolutionTool:OnWindowResize()
  local width, height = self:_GetDeviceResolution()
  if self.mmmApplied and self.device.x == width and self.device.y == height then
    return
  end
  self.mmmApplied = true
  self.device = CS.UnityEngine.Vector2(width, height)
  if GameConfig.LOG_LEVEL <= 2 then
    Log.Info("[ResolutionTool] OnWindowResize", self.device)
  end
  self:RefreshPerformanceResolution(true)
end
