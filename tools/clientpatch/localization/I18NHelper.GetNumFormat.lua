function I18NHelper.GetNumFormat(num)
  if not num then
    return
  end
  if 1000000 <= num then
    return Strings.Get("NUM_FORMAT_MILLION", {
      math.tointeger(math.floor(num / 1000000))
    })
  elseif 10000 <= num then
    return Strings.Get("NUM_FORMAT_THOUSAND", {
      math.tointeger(math.floor(num / 1000))
    })
  end
  return num
end
