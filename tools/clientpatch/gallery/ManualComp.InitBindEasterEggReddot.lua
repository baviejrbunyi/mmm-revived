function ManualComp:InitBindEasterEggReddot()
  local easterEggId, data = next(self.easterEgg2num)
  while easterEggId ~= nil do
    if dispatch_egg_hand_book_data.Get(easterEggId) then
      self._listenEasterEgg[easterEggId] = data
      self:ListenEasterEgg(easterEggId)
    end
    easterEggId, data = next(self.easterEgg2num, easterEggId)
  end
end
