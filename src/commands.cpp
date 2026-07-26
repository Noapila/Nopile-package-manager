#include "nopile.hpp"

#include <iostream>
#include <fstream>
#include <algorithm>

// ── help ──────────────────────────────────────────────────────────────────────

int cmd_help() {
    std::cout <<
"nopile " << NOPILE_VERSION << " - package manager for Noapila OS\n"
"\n"
"Usage:\n"
"  nopile install <package>        install a package (prebuilt binary)\n"
"  nopile install <package> --compile  build from source\n"
"  nopile remove  <package>        remove an installed package\n"
"  nopile update                   upgrade all installed packages\n"
"  nopile search  <term>           search available packages\n"
"  nopile help                     show this help\n"
"  nopile -i <file.tar.gz>         install from a local tarball\n"
"\n"
"Paths:\n"
"  packages    " << NOPILE_PKG_DIR    << "\n"
"  sources     " << NOPILE_SOURCES_DIR << "\n"
"  cache       " << NOPILE_CACHE_DIR  << "\n"
"  database    " << NOPILE_DB         << "\n";
    return 0;
}

// ── search ────────────────────────────────────────────────────────────────────

int cmd_search(const std::string& query) {
    auto entries = db_load();
    if (entries.empty()) {
        std::cerr << "[nopile] database empty or missing — run: nopile update\n";
        return 1;
    }

    bool found = false;
    for (const auto& e : entries) {
        if (e.name.find(query) != std::string::npos) {
            std::cout << e.name << "  " << e.version;
            if (!e.source_url.empty()) std::cout << "  [source available]";
            std::cout << '\n';
            found = true;
        }
    }
    if (!found) std::cerr << "[nopile] no results for: " << query << '\n';
    return found ? 0 : 1;
}

// ── install (core) ────────────────────────────────────────────────────────────

static int install_from_tarball(const fs::path& tarball,
                                const std::string& source_url,
                                bool compile) {
    // 1. Extract tarball into a temp dir
    fs::path tmp = fs::path(NOPILE_CACHE_DIR) / ("extract-" + tarball.stem().string());
    fs::remove_all(tmp);
    if (!tar_extract(tarball, tmp)) {
        std::cerr << "[nopile] extraction failed\n";
        return 1;
    }

    // 2. Read the .nopile inside the tarball
    fs::path nopile_in;
    for (auto& e : fs::recursive_directory_iterator(tmp))
        if (e.path().extension() == ".nopile") { nopile_in = e.path(); break; }

    if (nopile_in.empty()) {
        std::cerr << "[nopile] no .nopile found inside tarball\n";
        return 1;
    }

    NopileFile pkg = nopile_parse(nopile_in);
    pkg.source = source_url;

    if (!compile) {
        // ── Binary install ────────────────────────────────────────────────────

        // Find install.nopbuild
        fs::path build_file;
        for (auto& e : fs::recursive_directory_iterator(tmp))
            if (e.path().filename() == "install.nopbuild")
                { build_file = e.path(); break; }

        if (build_file.empty()) {
            std::cerr << "[nopile] no install.nopbuild found\n";
            return 1;
        }

        // Find package/ root
        fs::path pkg_root = build_file.parent_path() / "package";
        if (!fs::exists(pkg_root)) pkg_root = tmp;

        // Run nopbuild instructions
        std::cout << "[nopile] installing " << pkg.name << " " << pkg.version << "...\n";
        if (!nopbuild_run(build_file, pkg_root)) {
            std::cerr << "[nopile] installation failed\n";
            return 1;
        }

        // Compute md5 for every installed file (from [files] in .nopile)
        for (auto& file : pkg.files)
            file.md5 = md5_file(fs::path(file.path));

        // Install services
        for (const auto& svc : pkg.services) {
            // Check if this init is installed
            fs::path init_dir = fs::path("/usr/lib/init") / svc.init_name;
            if (!fs::exists(init_dir)) continue;

            // Find service files in tarball
            fs::path svc_src = build_file.parent_path() / "service" / svc.init_name;
            if (!fs::exists(svc_src)) continue;

            for (const auto& dest_path : svc.paths) {
                fs::path filename = fs::path(dest_path).filename();
                fs::path src_file = svc_src / filename;
                if (!fs::exists(src_file)) continue;
                fs::create_directories(fs::path(dest_path).parent_path());
                fs::copy_file(src_file, fs::path(dest_path),
                              fs::copy_options::overwrite_existing);
            }

            // Tell noapila-init to enable the service
            std::string enable_cmd = "noapila-init enable " + pkg.name + " 2>/dev/null || true";
            std::system(enable_cmd.c_str());
        }
    }
    else {
        // ── Compile install ───────────────────────────────────────────────────

        fs::path build_dir = fs::path(NOPILE_SOURCES_DIR) / pkg.name;
        fs::create_directories(build_dir);

        // Copy sources there
        for (auto& e : fs::directory_iterator(tmp))
            fs::copy(e.path(), build_dir / e.path().filename(),
                     fs::copy_options::recursive | fs::copy_options::overwrite_existing);

        // Find install.nopbuild (compile version)
        fs::path build_file;
        for (auto& e : fs::recursive_directory_iterator(build_dir))
            if (e.path().filename() == "install.nopbuild")
                { build_file = e.path(); break; }

        if (build_file.empty()) {
            std::cerr << "[nopile] no install.nopbuild found for compile\n";
            return 1;
        }

        std::cout << "[nopile] building " << pkg.name << " " << pkg.version
                  << " in " << build_dir << "...\n";

        // DESTDIR install: everything goes into build_dir/destdir/
        // e.g. /var/lib/nopile/sources/bash/destdir/usr/bin/bash
        fs::path destdir = build_dir / "destdir";
        fs::create_directories(destdir);

        if (!nopbuild_run(build_file, build_dir)) {
            std::cerr << "[nopile] build failed\n";
            return 1;
        }

        // Scan destdir recursively to find all installed files
        // Strip the destdir prefix to get the real system path
        // e.g. destdir/usr/bin/bash  ->  /usr/bin/bash
        pkg.files.clear();
        for (auto& e : fs::recursive_directory_iterator(destdir)) {
            if (!e.is_regular_file()) continue;

            fs::path rel = fs::relative(e.path(), destdir);
            fs::path sys_path = fs::path("/") / rel;

            // Copy file to real system path
            fs::create_directories(sys_path.parent_path());
            fs::copy_file(e.path(), sys_path,
                          fs::copy_options::overwrite_existing);

            InstalledFile f;
            f.path = sys_path.string();
            f.md5  = md5_file(sys_path);
            pkg.files.push_back(f);
        }
    }

    // Register the package
    fs::create_directories(NOPILE_PKG_DIR);
    fs::path reg = nopile_path(pkg.name);
    nopile_write(reg, pkg);

    std::cout << "[nopile] " << pkg.name << " " << pkg.version
              << " installed\n";

    fs::remove_all(tmp);
    return 0;
}

// ── install (public) ──────────────────────────────────────────────────────────

int cmd_install(const std::string& name, bool compile,
                bool local, const fs::path& local_path) {

    if (local) {
        // nopile -i /path/to/file.tar.gz
        if (!fs::exists(local_path)) {
            std::cerr << "[nopile] file not found: " << local_path << '\n';
            return 1;
        }
        return install_from_tarball(local_path, "local", compile);
    }

    // Lookup in DB
    auto entry = db_find(name);
    if (!entry) {
        std::cerr << "[nopile] package not found: " << name
                  << " (try: nopile update)\n";
        return 1;
    }

    if (nopile_exists(name)) {
        std::cout << "[nopile] " << name << " is already installed\n";
        return 0;
    }

    std::string url = compile ? entry->source_url : entry->binary_url;
    std::string expected_md5 = compile ? entry->source_md5 : entry->binary_md5;

    if (url.empty()) {
        std::cerr << "[nopile] no "
                  << (compile ? "source" : "binary")
                  << " available for: " << name << '\n';
        return 1;
    }

    // Download
    fs::path archive = fs::path(NOPILE_CACHE_DIR) / (name + ".tar.gz");
    std::cout << "[nopile] downloading " << name << "...\n";
    if (!net_download(url, archive)) {
        std::cerr << "[nopile] download failed\n";
        return 1;
    }

    // Verify md5
    if (!expected_md5.empty()) {
        std::string got = md5_file(archive);
        if (got != expected_md5) {
            std::cerr << "[nopile] md5 mismatch for " << name << '\n';
            std::cerr << "  expected: " << expected_md5 << '\n';
            std::cerr << "  got:      " << got << '\n';
            fs::remove(archive);
            return 1;
        }
    }

    return install_from_tarball(archive, url, compile);
}

// ── remove ────────────────────────────────────────────────────────────────────

int cmd_remove(const std::string& name) {
    if (!nopile_exists(name)) {
        std::cerr << "[nopile] not installed: " << name << '\n';
        return 1;
    }

    NopileFile pkg = nopile_parse(nopile_path(name));

    std::cout << "[nopile] removing " << pkg.name << " " << pkg.version << "...\n";

    // Remove each file after md5 check
    for (const auto& file : pkg.files) {
        fs::path p(file.path);
        if (!fs::exists(p)) continue;

        std::string actual = md5_file(p);
        if (actual != file.md5) {
            std::cerr << "[nopile] skipping modified file: " << file.path << '\n';
            continue;
        }
        fs::remove(p);
    }

    // Remove service files
    for (const auto& svc : pkg.services) {
        for (const auto& sp : svc.paths) {
            fs::path p(sp);
            if (fs::exists(p)) fs::remove(p);
        }
    }

    // Remove .nopile
    fs::remove(nopile_path(name));

    std::cout << "[nopile] " << name << " removed\n";
    return 0;
}

// ── update ────────────────────────────────────────────────────────────────────

int cmd_update() {
    // 1. Refresh DB
    std::cout << "[nopile] fetching package database...\n";
    if (!db_download()) {
        std::cerr << "[nopile] failed to download nopile.db\n";
        return 1;
    }

    // 2. Check installed packages
    fs::path dir(NOPILE_PKG_DIR);
    if (!fs::exists(dir)) {
        std::cout << "[nopile] no packages installed\n";
        return 0;
    }

    int upgraded = 0;
    for (auto& entry : fs::directory_iterator(dir)) {
        if (entry.path().extension() != ".nopile") continue;
        try {
            NopileFile local = nopile_parse(entry.path());
            auto remote = db_find(local.name);
            if (!remote) continue;
            if (remote->version != local.version) {
                std::cout << "[nopile] upgrading " << local.name
                          << " " << local.version
                          << " -> " << remote->version << '\n';
                fs::remove(nopile_path(local.name));
                cmd_install(local.name, false, false, {});
                ++upgraded;
            }
        } catch (...) {}
    }

    if (upgraded == 0) std::cout << "[nopile] all packages up to date\n";
    return 0;
}
