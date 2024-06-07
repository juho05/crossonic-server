import logging
import sys
import os
import soco
from queue import Empty
import time
import json

from flask import Flask
from flask import jsonify
from flask import request

from simple_websocket import Server, ConnectionClosed

log = logging.getLogger('werkzeug')
log.setLevel(logging.WARNING)

app = Flask(__name__)
logger = logging.getLogger("crossonic-sonos")
logger.setLevel(logging.DEBUG)
ch = logging.StreamHandler()
ch.setLevel(logging.DEBUG)
ch.setFormatter(logging.Formatter('%(asctime)s [%(levelname)s] %(message)s'))
logger.addHandler(ch)


current_queue_index = 0

@app.route("/ping")
def ping():
  logger.debug("/ping")
  return "crossonic-sonos", 200

@app.post("/getDevices")
def devices():
  devices = soco.discovery.scan_network(scan_timeout=1)
  result = []
  for d in devices:
    result.append({
      "name": d.player_name,
      "ip_addr": d.ip_address
    })
  logger.debug("/getDevices: found %d device(s)", len(devices))
  return jsonify(result), 200

@app.post("/<ip_addr>/stop")
def stop(ip_addr):
  zone = soco.SoCo(ip_addr)
  zone.stop()
  zone.clear_queue()
  logger.debug("/%s/stop", ip_addr)
  return "", 200

@app.post("/<ip_addr>/setCurrent")
def set_current(ip_addr):
  global current_queue_index
  uri = request.json.get("uri")
  if uri is None:
    return "missing uri field", 400
  nextURI = request.json.get("next_uri")
  zone = soco.SoCo(ip_addr)
  zone.play_mode = "NORMAL"
  current_queue_index = zone.queue_size
  zone.add_uri_to_queue(uri)
  if nextURI is not None:
    zone.add_uri_to_queue(nextURI)
  else:
    nextURI = ""
  zone.play_from_queue(current_queue_index, False)
  logger.debug("/%s/setCurrent: current=%s; next=%s", ip_addr, uri, nextURI)
  time.sleep(1)
  return "", 200

@app.post("/<ip_addr>/setNext")
def set_next(ip_addr):
  uri = request.json.get("uri")
  zone = soco.SoCo(ip_addr)
  current_index = int(zone.get_current_track_info()['playlist_position'])-1
  length = zone.queue_size
  if current_index+1 < length:
    zone.remove_from_queue(current_index+1)
  if uri is not None and len(uri) > 0:
    zone.add_uri_to_queue(uri)
  else:
    uri = ""
  logger.debug("/%s/setNext: next=%s", ip_addr, uri)
  return "", 200

@app.post("/<ip_addr>/getPosition")
def get_position(ip_addr):
  zone = soco.SoCo(ip_addr)
  pos = zone.get_current_track_info()["position"]
  parts = pos.split(":")
  if len(parts) == 3:
    seconds = int(parts[0])*60*60+int(parts[1])*60+int(parts[2])
  else:
    seconds = 0
  logger.debug("/%s/getPosition: %ds", ip_addr, seconds)
  return jsonify({"seconds": seconds}), 200

@app.post("/<ip_addr>/play")
def play(ip_addr):
  soco.SoCo(ip_addr).play()
  logger.debug("/%s/play", ip_addr)
  return "", 200

@app.post("/<ip_addr>/pause")
def pause(ip_addr):
  soco.SoCo(ip_addr).pause()
  logger.debug("/%s/pause", ip_addr)
  return "", 200

@app.post("/<ip_addr>/getVolume")
def get_volume(ip_addr):
  zone = soco.SoCo(ip_addr)
  volume = zone.volume
  logger.debug("/%s/getVolume: %d", ip_addr, volume)
  return jsonify({
    "volume": volume
  }), 200

@app.post("/<ip_addr>/setVolume")
def set_volume(ip_addr):
  volume = request.json.get("volume")
  if volume is None:
    return "missing volume field", 400
  zone = soco.SoCo(ip_addr)
  zone.volume = int(volume)
  logger.debug("/%s/setVolume: %d", ip_addr, int(volume))
  return "", 200

@app.route("/<ip_addr>/events", websocket=True)
def events(ip_addr):
  global current_queue_index
  speaker = soco.SoCo(ip_addr)
  try:
    sub = speaker.avTransport.subscribe(auto_renew=True)
  except Exception as e:
    logger.error("/%s/events: %s", ip_addr, e)
    return

  ws = Server.accept(request.environ)
  logger.debug("/%s/events: listening for events...", ip_addr)
  try:
    current_state = ""
    while True:
        try:
          event = sub.events.get(timeout=1.0)
          state = event.variables["transport_state"]
          queue_index = int(event.variables["current_track"])-1
          if queue_index > current_queue_index:
            print("advance")
            ws.send(json.dumps({
              "state": "advance"
            }))
            current_queue_index = queue_index
          if current_state != state:
            current_state = state
            print(state)
            ws.send(json.dumps({
              "state": state,
            }))
          ws.receive(0)
        except Empty:
          ws.receive(0)
          pass
        except KeyboardInterrupt:
          break
  except ConnectionClosed:
    logger.debug("/%s/events: websocket closed", ip_addr)
    pass
  except Exception as e:
    logger.error("/%s/events: %s", ip_addr, e)

  speaker.stop()
  sub.unsubscribe()
  logger.debug("/%s/events: sonos subscription canceled", ip_addr)
  if ws.connected:
    logger.debug("/%s/events: websocket closed", ip_addr)
    ws.close()
  return "", 200

if __name__ == "__main__":
  host = os.environ.setdefault("HOST", "127.0.0.1")
  port = os.environ.setdefault("PORT", "8257")
  logger.info("Listening on %s:%s...", host, port)
  app.run(host=host, port=port, debug=False)