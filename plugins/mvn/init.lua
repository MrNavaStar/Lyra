local lyra = require("lyra")
local http = require("http")
local re = require("re")
local xml = require("xml")

local pattern = "([^: ]+):([^: ]+)(:([^: ]*)(:([^: ]+))?)?:([^: ]+)"

local function parse_artifact(coordinate)
    local a, b, _, c, _, d, e = re.match(coordinate, pattern)
    return a, b, c, d, e
end

local function get_pom(repo, group, id, version)
   local pom_url = repo .. "/" .. group:gsub("%.", "/") .. "/" .. id .. "/" .. version .. "/" .. id .. "-" .. version .. ".pom"

    local req, err = http.get(pom_url)
    if err or req["status"] ~= 200 then
        return nil, err
    end

    return pom_url:gsub("%.pom", ""), xml.decode(req["body"])
end

local function resolve_artifact(repos, group, id, version)
    local deps = {}
    if repos == nil or group == nil or id == nil or version == nil then
        return deps
    end

    for i = 1, #repos do
        local url, pom, err = get_pom(repos[i], group, id, version)
        if err or url == nil or pom == nil or pom["project"] == nil then
            goto continue
        end

        deps[group .. ":" .. id] = {
            ["Version"] = version,
            ["Main"] = url .. ".jar",
            ["Sources"] = url .. "-sources.jar",
            ["Docs"] = url .. "-javadoc.jar",
        }

        -- Use pretty name if available in pom
        local name = pom["project"]["name"]
        if name ~= nil then
            deps[group .. ":" .. id]["Name"] = name
        end
        
        -- Resolve artifact dependencies
        local artifact_deps = pom["project"]["dependencies"]
        if artifact_deps == nil then
            goto continue
        end

        for _, dep in ipairs(artifact_deps["dependency"]) do
            local indirect = resolve_artifact(repos, dep["groupId"], dep["artifactId"], dep["version"])
            for key, val in pairs(indirect) do
                deps[key] = val
            end
        end
        ::continue::
    end
    return deps
end

local function mvn_resolver(repos, coordinate)
    local group, id, extension, classifier, version = parse_artifact(coordinate)
    return resolve_artifact(repos, group, id, version)
end

lyra.export_function("mvn", "parse_artifact", parse_artifact)
lyra.add_dependency_resolver(mvn_resolver)
