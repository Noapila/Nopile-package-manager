#include "nopile.hpp"
#include <iostream>
#include <cstdlib>
#include <fstream>

// ── Internal helpers ───────────────────────────────────────────────────────────

static int build_from_source(const PackageMeta& meta, bool verbose) {
    const fs::path build_dir = fs::path(NOPILE_CACHE_DIR) / ("build-" + meta.name);
    fs::create_directories(build_dir);

    // 1. Download sources
    print_info("Downloading sources: " + meta.source_url);
    fs::path src_archive = build_dir / (meta.name + "-src.tar.gz");
    if (!download_file(meta.source_url, src_archive)) {
        print_error("Failed to download sources.");
        return 1;
    }

    // 2. Extract
    print_info("Extracting...");
    std::string tar_cmd = "tar -xzf " + src_archive.string()
                        + " -C " + build_dir.string()
                        + (verbose ? " -v" : " -q");
    if (std::system(tar_cmd.c_str()) != 0) {
        print_error("Extraction failed.");
        return 1;
    }

    // 3. Find extracted directory (first subdirectory)
    fs::path src_dir;
    for (auto& e : fs::directory_iterator(build_dir))
        if (e.is_directory()) { src_dir = e.path(); break; }

    if (src_dir.empty()) {
        print_error("Source directory not found after extraction.");
        return 1;
    }

    // 4. Configure + build + install according to build_system
    int ret = 0;
    const std::string flags = meta.configure_flags.empty()
                            ? "--prefix=/usr"
                            : meta.configure_flags;
    const std::string nproc = "$(nproc)";
    const std::string v     = verbose ? "V=1" : "";

    if (meta.build_system == "cmake") {
        print_info("Configuring with CMake...");
        fs::path bdir = src_dir / "_build";
        fs::create_directories(bdir);
        ret = std::system(("cmake -S " + src_dir.string()
                         + " -B " + bdir.string()
                         + " " + flags).c_str());
        if (!ret) {
            print_info("Building...");
            ret = std::system(("cmake --build " + bdir.string()
                             + " -j" + nproc).c_str());
        }
        if (!ret) {
            print_info("Installing...");
            ret = std::system(("cmake --install " + bdir.string()).c_str());
        }
    }
    else if (meta.build_system == "meson") {
        print_info("Configuring with Meson...");
        fs::path bdir = src_dir / "_build";
        ret = std::system(("meson setup " + bdir.string()
                         + " " + src_dir.string()
                         + " " + flags).c_str());
        if (!ret) {
            print_info("Building...");
            ret = std::system(("ninja -C " + bdir.string()
                             + " -j" + nproc).c_str());
        }
        if (!ret) {
            print_info("Installing...");
            ret = std::system(("ninja -C " + bdir.string() + " install").c_str());
        }
    }
    else {
        // autotools or plain make (default)
        print_info("Configuring with Autotools...");
        ret = std::system(("cd " + src_dir.string()
                         + " && ./configure " + flags).c_str());
        if (!ret) {
            print_info("Building...");
            ret = std::system(("make -C " + src_dir.string()
                             + " -j" + nproc + " " + v).c_str());
        }
        if (!ret) {
            print_info("Installing...");
            ret = std::system(("make -C " + src_dir.string() + " install").c_str());
        }
    }

    return ret;
}

static int install_binary(const PackageMeta& meta, bool verbose) {
    const fs::path cache = fs::path(NOPILE_CACHE_DIR);
    fs::path archive = cache / (meta.name + ".tar.gz");

    std::string url = std::string(NOPILE_REPO_URL)
                    + "/packages/" + meta.name + "/" + meta.name + ".tar.gz";

    print_info("Downloading binary...");
    if (!download_file(url, archive)) {
        print_error("Failed to download binary package.");
        return 1;
    }

    print_info("Installing...");
    std::string cmd = "tar -xzf " + archive.string() + " -C /"
                    + (verbose ? " -v" : " -q");
    return std::system(cmd.c_str());
}

// ── install command ────────────────────────────────────────────────────────────

int cmd_install(const std::string& name, const InstallOptions& opts) {
    // Already installed?
    if (is_installed(name) && !opts.force) {
        print_info(name + " is already installed. Use --force to reinstall.");
        return 0;
    }

    // Fetch metadata
    auto meta_opt = fetch_package_meta(name);
    if (!meta_opt) return 1;
    PackageMeta meta = *meta_opt;

    // Resolve dependencies
    if (!opts.no_deps && !meta.depends.empty()) {
        for (const auto& dep : meta.depends) {
            if (!is_installed(dep)) {
                print_info("Installing dependency: " + dep);
                int r = cmd_install(dep, opts);
                if (r != 0) {
                    print_error("Failed to install dependency: " + dep);
                    return 1;
                }
            }
        }
    }

    // Build or install binary
    int ret = 0;
    if (opts.compile) {
        if (meta.source_url.empty()) {
            print_error("No source URL available for " + name + ".");
            return 1;
        }
        print_ok("Compile mode enabled for " + name);
        ret = build_from_source(meta, opts.verbose);
    } else {
        ret = install_binary(meta, opts.verbose);
    }

    if (ret != 0) {
        print_error("Installation of " + name + " failed.");
        return ret;
    }

    // Register package
    fs::create_directories(NOPILE_PKG_DIR);
    write_nopile_file(pkg_file_path(name), meta);

    print_ok(name + " " + meta.version + " installed successfully.");
    return 0;
}
