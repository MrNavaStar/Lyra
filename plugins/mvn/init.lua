local lyra = require("lyra")
local re = require("re")
local http = require("http")
local xml = require("xml")

local pattern = "([^: ]+):([^: ]+)(:([^: ]*)(:([^: ]+))?)?:([^: ]+)"

local function mvn_resolver(repos, coordinate)
    local dependencies = {}
    local _, group, id, _, extension, _, classifier, version = re.match(coordinate, pattern)

    for i = 1, #repos do
        local pom_url = group:gsub("%.", "/") .. "/" .. id .. "/" .. version .. "/" .. id .. "-" .. version .. ".pom"

        local req, err = http.get(repos[i] .. "/" .. pom_url)
        if err or req["status_code"] ~= 200 then
            goto continue
        end

        local pom, err = xml.decode(req["body"])
        if err then
            error(err)
        end

        print(pom)
        print(err)

        table.insert(dependencies, pom_url:gsub("%.pom", ".jar"))

        for _, dependency in ipairs(pom["dependencies"]) do
            dependencies = {table.unpack(dependencies), table.unpack(mvn_resolver(repos, dependency["groupId"] .. ":" + dependency["artifactId"] .. ":" .. dependency["version"]))}
        end
        ::continue::
    end

    print(dependencies)
    return dependencies
end

lyra.add_dependency_resolver(mvn_resolver)
