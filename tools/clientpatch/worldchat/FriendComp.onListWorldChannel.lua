function FriendComp:onListWorldChannel(result)
  local _result = json.decode(result)
  local helper = MC.UniMatrixHelper
  if not _result or _result.error_code ~= 0 or _result.data == nil or helper._subChannelId == nil then
    return
  end
  local list = _result.data.messages or {}
  local append = _result.data.append and true or false
  if append and helper._channel == nil then
    return
  end
  local messages = append and helper._channel.messages or {}
  for i = 1, #list do
    local text = list[i]
    local message = {
      ToSerializeString = function()
        return text
      end
    }
    messages[#messages + 1] = message
    if append then
      MC.EventUtil.Dispatch(EventDef.GLOBAL_CHAT_MESSAGE, {message = message})
    end
  end
  if not append then
    helper._channel = {
      channelInfo = {},
      messages = messages,
      counterPartChannelInfo = {}
    }
    MC.EventUtil.Dispatch(EventDef.ON_GET_CHANNEL_INFO, helper._channel.channelInfo, messages, helper._channel.counterPartChannelInfo)
  end
end
