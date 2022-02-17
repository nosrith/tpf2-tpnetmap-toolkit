require "gui"

local outPath = "map.yaml"

local function writeWorld(f)
    local worldEntity = api.engine.util.getWorld()
    local bbox = api.engine.getComponent(worldEntity, api.type.ComponentType.BOUNDING_VOLUME).bbox
    f:write(string.format(
        "{min: {x: %f,y: %f,z: %f},max: {x: %f,y: %f,z: %f}}", 
        bbox.min.x, bbox.min.y, bbox.min.z, 
        bbox.max.x, bbox.max.y, bbox.max.z
    ))
end

local function writeTowns(f)
    f:write("[")
    local t2cMap = api.engine.system.townBuildingSystem.getTown2personCapacitiesMap()
    local comma = ""
    for town in pairs(t2cMap) do
        local name = api.engine.getComponent(town, api.type.ComponentType.NAME).name
        local bbox = api.engine.getComponent(town, api.type.ComponentType.BOUNDING_VOLUME).bbox
        f:write(string.format(
            "%s{id: %d,name: \"%s\",x: %f,y: %f,z: %f}",
            comma, town, name, (bbox.min.x + bbox.max.x) / 2, (bbox.min.y + bbox.max.y) / 2, (bbox.min.z + bbox.max.z) / 2 
        ))
        comma = ","
    end
    f:write("]")
end

local function writeStations(f)
    f:write("[")
    local visitedGroup = {}
    local comma = ""
    api.engine.system.stationSystem.forEach(function(stationEntity)
        local groupEntity = api.engine.system.stationGroupSystem.getStationGroup(stationEntity)
        if visitedGroup[groupEntity] then
            return
        else
            visitedGroup[groupEntity] = true
        end

        local groupComp = api.engine.getComponent(groupEntity, api.type.ComponentType.STATION_GROUP)
        local nameComp = api.engine.getComponent(groupEntity, api.type.ComponentType.NAME)

        local x, y, z, c = 0, 0, 0, 0
        local cargo = false
        local hasTrack = false
        for _, stationEntityInGroup in ipairs(groupComp.stations) do
            local stationCompInGroup = api.engine.getComponent(stationEntityInGroup, api.type.ComponentType.STATION)
            cargo = cargo or stationCompInGroup.cargo
            local validPosition = false
            for _, terminal in ipairs(stationCompInGroup.terminals) do
                local nodeEntity = terminal.vehicleNodeId.entity

                local nodeComp = api.engine.getComponent(nodeEntity, api.type.ComponentType.BASE_NODE)
                if nodeComp then
                    x = x + nodeComp.position.x
                    y = y + nodeComp.position.y
                    z = z + nodeComp.position.z
                    c = c + 1
                    validPosition = true
                end
                
                local trackEdgeEntities = api.engine.system.streetSystem.getNode2TrackEdgeMap()[nodeEntity]
                if trackEdgeEntities then
                    hasTrack = true
                end
            end
            if not validPosition then
                boundComp = api.engine.getComponent(stationEntityInGroup, api.type.ComponentType.BOUNDING_VOLUME)
                x = x + (boundComp.bbox.min.x + boundComp.bbox.max.x) / 2
                y = y + (boundComp.bbox.min.y + boundComp.bbox.max.y) / 2
                z = z + (boundComp.bbox.min.z + boundComp.bbox.max.z) / 2
                c = c + 1
            end
        end 

        if c > 0 then
            f:write(string.format(
                "%s{id: %d,name: \"%s\",cargo: %s,hasTrack: %s,x: %f,y: %f,z: %f}", 
                comma, groupEntity, nameComp.name, cargo, hasTrack, x / c, y / c, z / c
            ))
            comma = ","
        end
    end)
    f:write("]")
end

local function collectPath(node, edge, edgeDesc, n2eMap, stack, visit, descFunc)
    local path = {}
    while 1 do
        local edgeComp = api.engine.getComponent(edge, api.type.ComponentType.BASE_EDGE)
        node = edgeComp.node0 == node and edgeComp.node1 or edgeComp.node0
        table.insert(path, node)
        if visit[node] then
            return path
        end
        visit[node] = true
        local connEdges = {}
        for _, connEdge in ipairs(n2eMap[node]) do
            if connEdge ~= edge and descFunc(connEdge) then
                table.insert(connEdges, connEdge)
            end
        end
        if #connEdges == 0 then
            return path
        elseif #connEdges == 1 then
            local connEdge = connEdges[1]
            local connEdgeDesc = descFunc(connEdge)
            if edgeDesc == connEdgeDesc then
                edge = connEdge
                edgeDesc = connEdgeDesc
            else
                table.insert(stack, node)
                table.insert(stack, connEdges)
                return path
            end
        else
            table.insert(stack, node)
            table.insert(stack, connEdges)
            return path
        end
    end
end

local function writeNetwork(f, descFunc)
    local n2eMap = api.engine.system.streetSystem.getNode2SegmentMap()

    local stack, visit = {}, {}
    local comma = ""
    f:write("{paths: [")
    for node, edges in pairs(n2eMap) do
        if not visit[node] then
            local targetEdges = {}
            for _, edge in ipairs(edges) do
                if descFunc(edge) then
                    table.insert(targetEdges, edge)
                end
            end
            if #targetEdges == 2 then
                visit[node] = true
                local edgeDesc1 = descFunc(targetEdges[1])
                local edgeDesc2 = descFunc(targetEdges[2])
                if edgeDesc1 == edgeDesc2 then
                    local path1 = collectPath(node, targetEdges[1], edgeDesc1, n2eMap, stack, visit, descFunc)
                    local path2 = collectPath(node, targetEdges[2], edgeDesc2, n2eMap, stack, visit, descFunc)
                    local revPath1 = {}
                    for i = #path1, 1, -1 do
                        table.insert(revPath1, path1[i])
                    end
                    f:write(string.format("%s{nodes: [%s,%d,%s],%s}", comma, table.concat(revPath1, ","), node, table.concat(path2, ","), edgeDesc1))
                    comma = ","
                else
                    table.insert(stack, node)
                    table.insert(stack, targetEdges)
                end
            elseif #targetEdges > 0 then
                visit[node] = true
                table.insert(stack, node)
                table.insert(stack, targetEdges)
            end
            while #stack > 0 do
                targetEdges = table.remove(stack)
                node = table.remove(stack)
                for _, edge in ipairs(targetEdges) do
                    local edgeDesc = descFunc(edge)
                    local path = collectPath(node, edge, edgeDesc, n2eMap, stack, visit, descFunc)
                    f:write(string.format("%s{nodes: [%d,%s],%s}", comma, node, table.concat(path, ","), edgeDesc))
                    comma = ","
                end
            end
        end
    end
    f:write("],nodes: {")
    comma = ""
    for node in pairs(visit) do
        local nodeComp = api.engine.getComponent(node, api.type.ComponentType.BASE_NODE)
        local position = nodeComp.position
        f:write(string.format("%s%d: {x: %f,y: %f,z: %f}", comma, node, position.x, position.y, position.z))
        comma = ","
    end
    f:write("}}")
end

local function writeTracks(f)
    writeNetwork(f, function(edge)
        local netComp = api.engine.getComponent(edge, api.type.ComponentType.TRANSPORT_NETWORK)
        local lane = netComp.edges[1]
        return lane.transportModes[api.type.enum.TransportMode.TRAIN + 1] == 1 and 
            string.format("speedLimit: %f", lane.speedLimit)
    end)
end

local function writeStreets(f)
    writeNetwork(f, function(edge)
        local netComp = api.engine.getComponent(edge, api.type.ComponentType.TRANSPORT_NETWORK)
        local numLanes, speedLimit, width = 0, 0, 0
        for _, lane in ipairs(netComp.edges) do
            if lane.transportModes[api.type.enum.TransportMode.CAR + 1] == 1 then
                numLanes = numLanes + 1
                speedLimit = lane.speedLimit
                width = lane.geometry.width
            end
        end
        return numLanes > 0 and string.format("numLanes: %d,speedLimit: %f,width: %f", numLanes, speedLimit, width)
    end)
end

function data()
    return {
        handleEvent = function(src, id, name, param)
            if id == "nosrith_tpnetmap_export" then
                local f = io.open(outPath, "w")
                f:write("{world: ")
                writeWorld(f)
                f:write(",towns: ")
                writeTowns(f)
                f:write(",stations: ")
                writeStations(f)
                f:write(",tracks: ")
                writeTracks(f)
                f:write(",streets: ")
                writeStreets(f)
                f:write("}")
                f:close()
            end
        end,
        guiUpdate = function()
            if not api.gui.util.getById("nosrith.tpnetmap.export.button") then
                local label = gui.textView_create("nosrith.tpnetmap.export.label", "TpNetMap Export")
                local button = gui.button_create("nosrith.tpnetmap.export.button", label)
                button:onClick(function()
                    local button = api.gui.util.getById("nosrith.tpnetmap.export.button")
                    button:setEnabled(false)
                    local cmd = api.type.SendScriptEvent.new()
                    cmd.fileName = "nosrith_tpnetmap_export.lua"
                    cmd.id = "nosrith_tpnetmap_export"
                    cmd.name = "nosrith_tpnetmap_export"
                    api.cmd.sendCommand(cmd, function()
                        local button = api.gui.util.getById("nosrith.tpnetmap.export.button")
                        button:setEnabled(true)
                    end)
                end)
                game.gui.boxLayout_addItem("gameInfo.layout", gui.component_create("nosrith.tpnetmap.export.sep", "VerticalLine").id)
                game.gui.boxLayout_addItem("gameInfo.layout", button.id)
            end
        end
    }
end
