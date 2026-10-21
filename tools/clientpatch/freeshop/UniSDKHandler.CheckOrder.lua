function UniSDKHandler:CheckOrder(productId, customInfo)
  local systemType = customInfo and customInfo.system_type or 1
  local systemId = customInfo and customInfo.system_id or 0
  MC.GameWorld.Avatar:CallServerMethod("passThroughCheck", productId, 1, systemType, systemId)
end
