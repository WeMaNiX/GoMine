package net

import (
	"gomine/interfaces"
	server2 "goraklib/server"
	"gomine/net/info"
	"goraklib/protocol"
	"gomine/players/handlers"
)

type GoRakLibAdapter struct {
	server interfaces.IServer
	rakLibServer *server2.GoRakLibServer
}

/**
 * Returns a new GoRakLib adapter to adapt to the RakNet server.
 */
func NewGoRakLibAdapter(server interfaces.IServer) *GoRakLibAdapter {
	var rakServer = server2.NewGoRakLibServer(server.GetName(), server.GetAddress(), server.GetPort())
	rakServer.SetMinecraftProtocol(info.LatestProtocol)
	rakServer.SetMinecraftVersion(info.GameVersionNetwork)
	rakServer.SetMaxConnectedSessions(server.GetMaximumPlayers())
	rakServer.SetDefaultGameMode("Creative")
	rakServer.SetMotd(server.GetMotd())

	return &GoRakLibAdapter{server, rakServer}
}

/**
 * Returns the GoRakLib server.
 */
func (adapter *GoRakLibAdapter) GetRakLibServer() *server2.GoRakLibServer {
	return adapter.rakLibServer
}

/**
 * Ticks the adapter
 */
func (adapter *GoRakLibAdapter) Tick() {
	go adapter.rakLibServer.Tick()

	for _, session := range adapter.rakLibServer.GetSessionManager().GetSessions() {
		go func(session *server2.Session) {
			for _, encapsulatedPacket := range session.GetReadyEncapsulatedPackets() {

				player, _ := adapter.server.GetPlayerFactory().GetPlayerBySession(session)

				batch := NewMinecraftPacketBatch(player, adapter.server.GetLogger())
				batch.Buffer = encapsulatedPacket.Buffer
				batch.Decode()

				for _, packet := range batch.GetPackets() {
					packet.DecodeHeader()
					packet.Decode()

					priorityHandlers := GetPacketHandlers(packet.GetId())

					var handled = false
					for _, h := range priorityHandlers {
						for _, handler := range h {
							if packet.IsDiscarded() {
								return
							}

							ret := handler.Handle(packet, player, session, adapter.server)
							if !handled {
								handled = ret
							}
						}
					}

					if !handled {
						adapter.server.GetLogger().Debug("Unhandled Minecraft packet with ID:", packet.GetId())
					}
				}
			}
		}(session)
	}

	for _, pk := range adapter.rakLibServer.GetRawPackets() {
		adapter.server.HandleRaw(pk)
	}

	for _, session := range adapter.rakLibServer.GetSessionManager().GetDisconnectedSessions() {
		player, _ := adapter.server.GetPlayerFactory().GetPlayerBySession(session)
		handler := handlers.NewDisconnectHandler()
		handler.Handle(player, session, adapter.server)
	}
}

func (adapter *GoRakLibAdapter) GetSession(address string, port uint16) *server2.Session {
	var session, _ = adapter.rakLibServer.GetSessionManager().GetSession(address, port)
	return session
}

func (adapter *GoRakLibAdapter) SendPacket(pk interfaces.IPacket, player interfaces.IPlayer, priority byte) {
	var b = NewMinecraftPacketBatch(player, adapter.server.GetLogger())
	b.AddPacket(pk)

	adapter.SendBatch(b, player.GetSession(), priority)
}

func (adapter *GoRakLibAdapter) SendBatch(batch interfaces.IMinecraftPacketBatch, session *server2.Session, priority byte) {
	session.SendConnectedPacket(batch, protocol.ReliabilityReliableOrdered, priority)
}

/**
 * Returns if a packet with the given ID is registered.
 */
func (adapter *GoRakLibAdapter) IsPacketRegistered(id int) bool {
	return IsPacketRegistered(id)
}

/**
 * Returns a new packet with the given ID and a function that returns that packet.
 */
func (adapter *GoRakLibAdapter) RegisterPacket(id int, function func() interfaces.IPacket) {
	RegisterPacket(id, function)
}

/**
 * Returns a new packet with the given ID.
 */
func (adapter *GoRakLibAdapter) GetPacket(id int) interfaces.IPacket {
	return GetPacket(id)
}

/**
 * Registers a new packet handler to listen for packets with the given ID.
 * Returns a bool indicating success.
 */
func (adapter *GoRakLibAdapter) RegisterPacketHandler(id int, handler interfaces.IPacketHandler, priority int) bool {
	return RegisterPacketHandler(id, handler, priority)
}

/**
 * Returns all packet handlers registered on the given ID.
 */
func (adapter *GoRakLibAdapter) GetPacketHandlers(id int) [][]interfaces.IPacketHandler {
	return GetPacketHandlers(id)
}

/**
 * Deletes all packet handlers listening for packets with the given ID, on the given priority.
 */
func (adapter *GoRakLibAdapter) DeregisterPacketHandlers(id int, priority int) {
	DeregisterPacketHandlers(id, priority)
}

/**
 * Deletes a registered packet with the given ID.
 */
func (adapter *GoRakLibAdapter) DeletePacket(id int) {
	DeregisterPacket(id)
}