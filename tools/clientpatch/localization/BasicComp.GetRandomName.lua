function BasicComp:GetRandomName()
  local randomNameData = random_name_data.Get(string.upper(MC.I18NHelper.LANGUAGE_EN))
  local index = math.random(1, #randomNameData.firstName)
  local firstName = randomNameData.firstName[index]
  local length = string.utf8len(firstName)
  local maxlength = constant_data.nick_name_length_limit_max
  local lastNames = {}
  for i = 1, #randomNameData.lastName do
    local name = randomNameData.lastName[i]
    if maxlength >= string.utf8len(name) + length then
      table.insert(lastNames, name)
    end
  end
  if 0 < #lastNames then
    return firstName .. lastNames[math.random(1, #lastNames)]
  end
  return firstName
end
