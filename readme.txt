zum bauen
1. auf eine shell des smarthome2.fritz.box gehen, dort ist go und die USB lib installiert

dann nach /home/pi/xcomfortd-go  wechseln

2. lokales build machen mit:   go build -buildvcs=false .

3. zum test deamon starten, nachdem in Homeassistant xcomfort gestoppt wurde
   sudo ./xcomfortd-go --verbose --hd -e -s tcp://smarthome2.fritz.box:1883
   sudo ./xcomfortd-go --verbose --hd -e -s tcp://pi:pi@homeassistant.fritz.box:1883


Zum Bauen für Homeassistant
===========================
auf Linux Laptop wechseln
     user: hjzimmer
     pw: Caro2001
ins Verzeichnis    /home/hjzimmer/xcomfort/xcomfortd-go/build  gehen
in docker hub einloggen
     docker login -u hajozi70
     pw: Caro2001!
dann die docker images crosscopilieren und uploaden mit
     make docker-builder-prepare
     make docker-builder-create
     make docker-build

auf dem Homeassistant liegt das addon in   /root/addons/temp/xcomfortd
dort ist das Dockerfile, welches HA nutzt, wenn das Addon installiert wird
	jedesmal das Addon deinstallieren und installieren, damit der Dockercontainer 
        die neuen Binaries aus docker hub nimmt

