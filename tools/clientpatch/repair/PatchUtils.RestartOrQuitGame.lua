function PatchUtils.RestartOrQuitGame()
  local ret, msg = xpcall(function()
    if CS.UnityEngine.Application.platform == CS.UnityEngine.RuntimePlatform.WindowsPlayer then
      local exePath = CS.System.IO.Path.GetDirectoryName(CS.UnityEngine.Application.dataPath)
      local launcherPath = string.format("%s/launcher.exe", exePath)
      if CS.System.IO.File.Exists(launcherPath) then
        local args = CS.System.Environment.GetCommandLineArgs()
        local restartArgs = {
          "--WaitTimeout=2000"
        }
        for i = 0, args.Length - 1 do
          if string.find(args[i], "--launcher") ~= nil then
            table.insert(restartArgs, args[i])
          end
        end
        mclua.StartProcess(launcherPath, restartArgs)
      end
      mclua.KillProcess(mclua.GetCurrentProcess())
    elseif CS.UnityEngine.Application.isEditor then
      CS.UnityEditor.EditorApplication.ExitPlaymode()
    else
      CS.UnityEngine.Application.Quit()
    end
  end, OnException)
end
