set windows-shell := ["powershell.exe", "-NoLogo", "-NoProfile", "-Command"]
set default-list

mirror $folder="":
    @go run ./tools/control mirror

configure $client $content="":
    @go run ./tools/control configure

rotate-keys $client:
    @go run ./tools/control rotate-keys

join $client $server:
    @go run ./tools/control join

run:
    @go run ./tools/control run

reset:
    @go run ./tools/control reset

pack-content $folder="":
    @go run ./tools/control pack-content
