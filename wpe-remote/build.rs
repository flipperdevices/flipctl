fn main() {
    pkg_config::probe_library("wpe-1.0")
        .expect("wpe-1.0 not found — install libwpe");
    pkg_config::probe_library("wpebackend-fdo-1.0")
        .expect("wpebackend-fdo-1.0 not found — install wpebackend-fdo");
    pkg_config::Config::new()
        .atleast_version("2.0")
        .probe("wpe-webkit-2.0")
        .expect("wpe-webkit-2.0 not found — sudo pacman -S wpewebkit");
    pkg_config::probe_library("wayland-server")
        .expect("wayland-server not found");
}
