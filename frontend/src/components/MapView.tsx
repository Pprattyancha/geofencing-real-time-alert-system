import { MapContainer, TileLayer, Polygon, CircleMarker, useMapEvents } from 'react-leaflet'
import type { Coordinate, Geofence, Vehicle } from '../types'

function ClickCapture({ onPoint }: {onPoint:(point:Coordinate)=>void}) {
  useMapEvents({ click(e) { onPoint([Number(e.latlng.lat.toFixed(6)), Number(e.latlng.lng.toFixed(6))]) } })
  return null
}
export default function MapView({ geofences, vehicles, onMapClick, draft }: {
  geofences:Geofence[]; vehicles:Vehicle[]; onMapClick?:(point:Coordinate)=>void; draft:Coordinate[]
}) {
  return <MapContainer center={[20.5937,78.9629]} zoom={5} scrollWheelZoom className="map">
    <TileLayer attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>' url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png" />
    {geofences.map(g => <Polygon key={g.id} positions={g.coordinates} pathOptions={{color:g.category==='restricted_zone'?'#f05d72':'#48c7a5', fillOpacity:.14, weight:2}} />)}
    {draft.length > 1 && <Polygon positions={draft} pathOptions={{color:'#8b7cff',dashArray:'7 6',fillOpacity:.08}} />}
    {onMapClick && <ClickCapture onPoint={onMapClick} />}
    {vehicles.map((v,i) => <CircleMarker key={v.id} center={[20.5937+(i*.01),78.9629+(i*.01)]} radius={7} pathOptions={{color:'#fff',fillColor:'#58a6ff',fillOpacity:1,weight:2}} />)}
  </MapContainer>
}