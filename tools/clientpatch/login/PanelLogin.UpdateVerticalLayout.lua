function PanelLogin:UpdateVerticalLayout()
  local top = self:FindChildByPath("top")
  local bottom = self:FindChildByPath("bottom")
  local trans = self:FindChildCompByPath("cont", typeof(u3d.RectTransform))
  local screenPos = MC.UIUtils.GetScreenPosition(bottom)
  if screenPos.y < 0 then
    trans:SetSizeWithCurrentAnchors(u3d.RectTransform.Axis.Vertical, 1080)
  else
    local height = top.transform.anchoredPosition.y - bottom.transform.anchoredPosition.y
    trans:SetSizeWithCurrentAnchors(u3d.RectTransform.Axis.Vertical, height)
  end
  if not self.passwordInputField then
    return
  end
  local scale = math.min(1, trans.rect.height / 900, trans.rect.width / 800)
  if scale <= 0 then
    return
  end
  local fields = {self.userNameInputField, self.passwordInputField}
  for index = 1, 2 do
    local field = fields[index]
    local rect = field.gameObject:GetComponent(typeof(u3d.RectTransform))
    rect.anchorMin = CS.UnityEngine.Vector2(0.5, 0.5)
    rect.anchorMax = CS.UnityEngine.Vector2(0.5, 0.5)
    rect.pivot = CS.UnityEngine.Vector2(0.5, 0.5)
    rect.sizeDelta = CS.UnityEngine.Vector2(560, 84)
    rect.localScale = CS.UnityEngine.Vector3(scale, scale, 1)
    rect.anchoredPosition = CS.UnityEngine.Vector2(0, (-50 - (index - 1) * 105) * scale)
    rect:SetAsLastSibling()
    local image = field.gameObject:GetComponent(typeof(CS.UnityEngine.UI.Image))
    if image ~= nil and not isnull(image) then
      image.color = CS.UnityEngine.Color(0, 0, 0, 0)
      image.raycastPadding = CS.UnityEngine.Vector4(0, 0, 0, 0)
    end
    local ok, failure = pcall(function()
      local template = self.mmmBoxTemplate
      if template == nil or isnull(template) then
        local all = rect.root:GetComponentsInChildren(typeof(CS.UnityEngine.UI.Image), true)
        local rank = 0
        for item = 0, all.Length - 1 do
          local entry = all[item]
          local name = entry.gameObject.name
          local value = 0
          if entry.sprite ~= nil and not isnull(entry.sprite) then
            if name == "img_black" then
              value = 3
            elseif name == "Template" or name == "CGBg" then
              value = 2
            elseif entry.type == CS.UnityEngine.UI.Image.Type.Sliced then
              value = 1
            end
          end
          if value > rank then
            rank = value
            template = entry
          end
        end
        self.mmmBoxTemplate = template
      end
      local background = rect:Find("mmm_box")
      if background == nil or isnull(background) then
        local backgroundObject = CS.UnityEngine.GameObject("mmm_box")
        backgroundObject.layer = field.gameObject.layer
        backgroundObject:AddComponent(typeof(u3d.RectTransform))
        backgroundObject:AddComponent(typeof(CS.UnityEngine.UI.Image))
        backgroundObject.transform:SetParent(rect, false)
        background = backgroundObject.transform
      end
      local old = rect:Find("mmm_background")
      if old ~= nil and not isnull(old) then
        CS.UnityEngine.Object.Destroy(old.gameObject)
      end
      local backgroundRect = background:GetComponent(typeof(u3d.RectTransform))
      backgroundRect.anchorMin = CS.UnityEngine.Vector2(0, 0)
      backgroundRect.anchorMax = CS.UnityEngine.Vector2(1, 1)
      backgroundRect.pivot = CS.UnityEngine.Vector2(0.5, 0.5)
      backgroundRect.sizeDelta = CS.UnityEngine.Vector2(0, 0)
      backgroundRect.anchoredPosition = CS.UnityEngine.Vector2(0, 0)
      backgroundRect.localScale = CS.UnityEngine.Vector3(1, 1, 1)
      background:SetAsFirstSibling()
      local backgroundImage = background:GetComponent(typeof(CS.UnityEngine.UI.Image))
      if template ~= nil and not isnull(template) then
        backgroundImage.sprite = template.sprite
        backgroundImage.material = template.material
        backgroundImage.type = template.sprite.border.x > 0 and CS.UnityEngine.UI.Image.Type.Sliced or CS.UnityEngine.UI.Image.Type.Simple
      end
      backgroundImage.color = CS.UnityEngine.Color(0, 0, 0, 0.6)
      backgroundImage.raycastTarget = false
      backgroundImage.enabled = true
      local file = io.open(CS.MCEngine.PathHelper.PersistentResPath .. "/mmm_login_field" .. index .. ".txt", "w")
      if file then
        file:write("template=" .. tostring(template and template.gameObject.name) .. " sprite=" .. tostring(template and template.sprite.name) .. "\n")
        file:close()
      end
    end)
    if not ok then
      pcall(function()
        local file = io.open(CS.MCEngine.PathHelper.PersistentResPath .. "/mmm_login_field" .. index .. ".txt", "w")
        file:write("failed: " .. tostring(failure) .. "\n")
        file:close()
      end)
    end
    local texts = {field.textComponent, field.placeholder:GetComponent(typeof(CS.UnityEngine.UI.Text))}
    for textIndex = 1, 2 do
      local text = texts[textIndex]
      text.rectTransform.anchorMin = CS.UnityEngine.Vector2(0, 0)
      text.rectTransform.anchorMax = CS.UnityEngine.Vector2(1, 1)
      text.rectTransform.sizeDelta = CS.UnityEngine.Vector2(-48, -16)
      text.rectTransform.anchoredPosition = CS.UnityEngine.Vector2(0, 0)
      text.fontSize = 28
      text.alignment = CS.UnityEngine.TextAnchor.MiddleLeft
      text.raycastTarget = false
      if textIndex == 1 then
        text.color = CS.UnityEngine.Color(1, 1, 1, 1)
      else
        text.color = CS.UnityEngine.Color(0.72, 0.76, 0.82, 1)
        text.text = index == 1 and "Username" or "Password"
      end
    end
  end
  local button = self._btnLogin:GetComponent(typeof(u3d.RectTransform))
  button.anchorMin = CS.UnityEngine.Vector2(0.5, 0.5)
  button.anchorMax = CS.UnityEngine.Vector2(0.5, 0.5)
  button.anchoredPosition = CS.UnityEngine.Vector2(0, -290 * scale)
  button.localScale = CS.UnityEngine.Vector3(scale, scale, 1)
  button:SetAsLastSibling()
  local logo = self.imgLogo.rectTransform
  logo.anchorMin = CS.UnityEngine.Vector2(0.5, 0.5)
  logo.anchorMax = CS.UnityEngine.Vector2(0.5, 0.5)
  logo.sizeDelta = CS.UnityEngine.Vector2(390, 360)
  logo.anchoredPosition = CS.UnityEngine.Vector2(0, 230 * scale)
  logo.localScale = CS.UnityEngine.Vector3(scale, scale, 1)
end
