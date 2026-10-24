function UniMatrixHelper:SendWorldMessage(message, chatMsgType)
  if self._subChannelId == nil or self._subChannelId == "" or MC.GameWorld.Avatar == nil then
    return
  end
  MC.GameWorld.Avatar:CallServerMethod("sendChat2Channel", Const.ChatChannelType.WORLD, chatMsgType, message, {
    channelId = self._subChannelId
  })
end
