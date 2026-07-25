#include "nopile.hpp"
#include <cstdlib>
#include <fstream>
#include <sstream>
#include <iostream>

// Uses curl (must be installed on the LFS system)

bool download_file(const std::string& url, const fs::path& dest) {
    fs::create_directories(dest.parent_path());
    std::string cmd = "curl -fsSL --retry 3 -o "
                    + dest.string() + " \"" + url + "\" 2>&1";
    int ret = std::system(cmd.c_str());
    return ret == 0;
}

// Downloads the .nopile metadata file from the remote repository.
// Expected repo layout:
//   packages/<name>/<name>.nopile       → metadata
//   packages/<name>/<name>.tar.gz       → prebuilt binary tarball
//   packages/<name>/<name>-src.tar.gz   → source tarball (for --compile)
std::optional<PackageMeta> fetch_package_meta(const std::string& name) {
    fs::path cache = fs::path(NOPILE_CACHE_DIR) / (name + ".nopile");
    std::string url = std::string(NOPILE_REPO_URL)
                    + "/packages/" + name + "/" + name + ".nopile";

    if (!download_file(url, cache)) {
        print_error("Package not found in repository: " + name);
        return std::nullopt;
    }

    try {
        return parse_nopile_file(cache);
    } catch (const std::exception& e) {
        print_error(e.what());
        return std::nullopt;
    }
}
