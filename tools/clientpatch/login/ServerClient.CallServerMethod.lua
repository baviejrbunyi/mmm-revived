function ServerClient:CallServerMethod(entityId, channelId, methodName, params, noTimeout)
  if self._gateClient ~= nil then
    if GameConfig.LOG_LEVEL <= 1 and methodName ~= "login" then
      Log.Trace("[C2S] CallServerMethod", mclua.MsgpackExtToString(entityId), methodName, table.tostring(params))
    end
    local checkFunc = self._checkCallServer[methodName]
    if checkFunc then
      local suc, r = mcpcall(checkFunc, self._checkCallServer, params)
      if suc and not r then
        return
      end
    end
    if self._gateClient:CallServerMethod(entityId, channelId, methodName, params) and not noTimeout then
      self:SetTimer(methodName)
    end
  end
end
