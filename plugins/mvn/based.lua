
local pattern = "([^: ]+):([^: ]+)(:([^: ]*)(:([^: ]+))?)?:([^: ]+)"
local group, id, _, extension, _, classifier, version = string.match("me.mrnavastar.protoweaver:fabric:1.3.15", pattern)
print(group)
print(id)
print(version)

lyra.test()
