function FriendComp:onJoinWorldChannel(result)
  local _result = json.decode(result)
  if _result and _result.error_code == 0 and _result.data then
    MC.UniMatrixHelper:SetChannelId(_result.data.channelId, _result.data.subChannelId)
    MC.EventUtil.Dispatch(EventDef.CONNECT_CHANNEL_RESULT, true, "CONNECT_WORLD_CHANNEL_FAIL")
    MC.GameWorld.Avatar:CallServerMethod("listWorldChannel", _result.data.subChannelId)
  else
    MC.EventUtil.Dispatch(EventDef.CONNECT_CHANNEL_RESULT, false, "CONNECT_WORLD_CHANNEL_FAIL")
  end
end
