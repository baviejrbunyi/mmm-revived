function UniMatrixHelper:JoinDivisionWorldChannel()
  local avatar = MC.GameWorld.Avatar
  if avatar == nil then
    return
  end
  if avatar.BasicComp:IsCNChildrenProtectEnabled() then
    MC.EventUtil.Dispatch(EventDef.CONNECT_CHANNEL_RESULT, false, "INVALID_IN_USER_CN_CHILDREN_MODE")
    return
  end
  local channelId = string.format("D%dEN%d@AS", tonumber(self._hostnum) or 0, tonumber(avatar.BasicComp:GetServerSet()) or 0)
  avatar:CallServerMethod("joinWorldChannel", channelId)
end
