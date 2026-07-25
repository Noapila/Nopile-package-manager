#include "nopile.hpp"
#include <iostream>
#include <fstream>
#include <iomanip>
#include <algorithm>
#include <cstdlib>

// ── remove ─────────────────────────────────────────────────────────────────────

int cmd_remove(const std::string& name, bool purge) {
    if (!is_installed(name)) {
        print_error(name + " is not installed.");
        return 1;
    }

    PackageMeta meta = parse_nopile_file(pkg_file_path(name));

    for (const auto& file : meta.files) {
        fs::path p(file);
        if (fs::exists(p)) {
            fs::remove(p);
            if (purge)
                std::cout << "  removed: " << file << '\n';
        }
    }

    fs::remove(pkg_file_path(name));
    print_ok(name + " removed.");
    return 0;
}

// ── list ───────────────────────────────────────────────────────────────────────

int cmd_list() {
    fs::path dir(NOPILE_PKG_DIR);
    if (!fs::exists(dir)) {
        print_info("No packages installed.");
        return 0;
    }

    std::vector<PackageMeta> pkgs;
    for (auto& entry : fs::directory_iterator(dir)) {
        if (entry.path().extension() == ".nopile") {
            try { pkgs.push_back(parse_nopile_file(entry.path())); } catch (...) {}
        }
    }

    std::sort(pkgs.begin(), pkgs.end(),
              [](const PackageMeta& a, const PackageMeta& b){ return a.name < b.name; });

    if (pkgs.empty()) {
        print_info("No packages installed.");
        return 0;
    }

    std::cout << "\033[1m"
              << std::left
              << std::setw(24) << "Name"
              << std::setw(16) << "Version"
              << "Description"
              << "\033[0m\n";
    std::cout << std::string(72, '-') << '\n';

    for (const auto& p : pkgs) {
        std::string desc = p.description;
        if (desc.size() > 42) desc = desc.substr(0, 39) + "...";
        std::cout << std::left
                  << std::setw(24) << p.name
                  << std::setw(16) << p.version
                  << desc << '\n';
    }
    std::cout << '\n' << pkgs.size() << " package(s) installed.\n";
    return 0;
}

// ── info ───────────────────────────────────────────────────────────────────────

int cmd_info(const std::string& name) {
    if (!is_installed(name)) {
        auto meta_opt = fetch_package_meta(name);
        if (!meta_opt) {
            print_error("Package not found: " + name);
            return 1;
        }
        const auto& m = *meta_opt;
        std::cout << "\033[33m[not installed]\033[0m\n";
        std::cout << "Name        : " << m.name        << '\n';
        std::cout << "Version     : " << m.version     << '\n';
        std::cout << "Description : " << m.description << '\n';
        std::cout << "Arch        : " << m.arch        << '\n';
        if (!m.depends.empty()) {
            std::cout << "Depends     : ";
            for (size_t i = 0; i < m.depends.size(); ++i) {
                if (i) std::cout << ", ";
                std::cout << m.depends[i];
            }
            std::cout << '\n';
        }
        if (!m.source_url.empty())
            std::cout << "Source      : " << m.source_url << '\n';
        return 0;
    }

    PackageMeta m = parse_nopile_file(pkg_file_path(name));
    std::cout << "\033[32m[installed]\033[0m\n";
    std::cout << "Name        : " << m.name        << '\n';
    std::cout << "Version     : " << m.version     << '\n';
    std::cout << "Description : " << m.description << '\n';
    std::cout << "Maintainer  : " << m.maintainer  << '\n';
    std::cout << "Arch        : " << m.arch        << '\n';
    if (!m.depends.empty()) {
        std::cout << "Depends     : ";
        for (size_t i = 0; i < m.depends.size(); ++i) {
            if (i) std::cout << ", ";
            std::cout << m.depends[i];
        }
        std::cout << '\n';
    }
    std::cout << "Files       : " << m.files.size() << " file(s)\n";
    if (!m.source_url.empty())
        std::cout << "Source      : " << m.source_url << '\n';
    return 0;
}

// ── search ─────────────────────────────────────────────────────────────────────

int cmd_search(const std::string& query) {
    fs::path idx_cache = fs::path(NOPILE_CACHE_DIR) / "index.txt";
    std::string url = std::string(NOPILE_REPO_URL) + "/packages/index.txt";

    if (!download_file(url, idx_cache)) {
        print_error("Failed to fetch package index.");
        return 1;
    }

    std::ifstream f(idx_cache);
    std::string line;
    bool found = false;
    while (std::getline(f, line)) {
        if (line.find(query) != std::string::npos) {
            std::cout << line << '\n';
            found = true;
        }
    }
    if (!found) print_info("No results for: " + query);
    return 0;
}

// ── update ─────────────────────────────────────────────────────────────────────

int cmd_update() {
    print_info("Checking for updates...");
    fs::path dir(NOPILE_PKG_DIR);
    if (!fs::exists(dir)) {
        print_info("No packages installed.");
        return 0;
    }

    int updated = 0;
    for (auto& entry : fs::directory_iterator(dir)) {
        if (entry.path().extension() != ".nopile") continue;
        try {
            PackageMeta local = parse_nopile_file(entry.path());
            auto remote_opt = fetch_package_meta(local.name);
            if (!remote_opt) continue;
            if (remote_opt->version != local.version) {
                std::cout << "Upgrading: " << local.name
                          << "  " << local.version
                          << " -> " << remote_opt->version << '\n';
                InstallOptions opts;
                opts.force = true;
                cmd_install(local.name, opts);
                ++updated;
            }
        } catch (...) {}
    }

    if (updated == 0) print_ok("All packages are up to date.");
    else print_ok(std::to_string(updated) + " package(s) upgraded.");
    return 0;
}
