function ResolutionTool:_GetDeviceResolution()
  if CS.UnityEngine.Application.isMobilePlatform then
    if GameConfig.IS_ANDROID and CS.MCEngine.FeatureConfig.MobileMultiWindowMode == 1 and CS.MCEngine.Base.UTools.IsInMultiWindowMode() then
      local width, height = CS.MCEngine.Base.UTools.GetMultiWindowModeWidth(), CS.MCEngine.Base.UTools.GetMultiWindowModeHeight()
      if 0 < width and 0 < height then
        return width, height
      end
    end
    return CS.UnityEngine.Display.main.systemWidth, CS.UnityEngine.Display.main.systemHeight
  elseif GameConfig.IS_PC then
    if self.isFullScreen then
      return CS.UnityEngine.Display.main.systemWidth, CS.UnityEngine.Display.main.systemHeight
    end
    local width, height = CS.UnityEngine.Screen.width, CS.UnityEngine.Screen.height
    if width < 320 or height < 240 then
      return MC.GameSettingHelper.GetResolutionByString(CS.UnityEngine.PlayerPrefs.GetString("[game_setting]_ResolutionRatio"))
    end
    CS.UnityEngine.PlayerPrefs.SetString("[game_setting]_ResolutionRatio", string.format("%dx%d", width, height))
    return width, height
  else
    return CS.UnityEngine.Screen.width, CS.UnityEngine.Screen.height
  end
end
