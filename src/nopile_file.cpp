#include "nopile.hpp"

#include <fstream>
#include <sstream>
#include <stdexcept>

// ── .nopile format ────────────────────────────────────────────────────────────
//
//  name:    bash
//  version: 5.2.21
//  source:  https://...
//
//  [files]
//  /usr/bin/bash
//  d41d8cd98f00b204e9800998ecf8427e
//
//  /etc/bashrc
//  a3f5b2c1...
//
//  [services]
//  systemd:
//  /usr/lib/init/systemd/system/sshd.service
//
//  runit:
//  /usr/lib/init/runit/etc/sv/sshd
//  /usr/lib/init/runit/etc/runsvdir/default/sshd
// ─────────────────────────────────────────────────────────────────────────────

static std::string trim(const std::string& s) {
    size_t b = s.find_first_not_of(" \t\r\n");
    if (b == std::string::npos) return {};
    size_t e = s.find_last_not_of(" \t\r\n");
    return s.substr(b, e - b + 1);
}

fs::path nopile_path(const std::string& name) {
    return fs::path(NOPILE_PKG_DIR) / (name + ".nopile");
}

bool nopile_exists(const std::string& name) {
    return fs::exists(nopile_path(name));
}

NopileFile nopile_parse(const fs::path& path) {
    std::ifstream f(path);
    if (!f) throw std::runtime_error("Cannot open: " + path.string());

    NopileFile pkg;
    std::string section;
    std::string line;
    std::string current_init;
    InstalledFile cur_file;
    bool waiting_md5 = false;

    auto flush_file = [&]() {
        if (!cur_file.path.empty()) {
            pkg.files.push_back(cur_file);
            cur_file = {};
            waiting_md5 = false;
        }
    };

    while (std::getline(f, line)) {
        line = trim(line);
        if (line.empty()) { if (waiting_md5) flush_file(); continue; }
        if (line[0] == '#') continue;

        // Section header
        if (line[0] == '[') {
            flush_file();
            section = line.substr(1, line.size() - 2);
            current_init.clear();
            continue;
        }

        if (section.empty()) {
            // Header fields
            auto colon = line.find(':');
            if (colon == std::string::npos) continue;
            std::string key = trim(line.substr(0, colon));
            std::string val = trim(line.substr(colon + 1));
            if      (key == "name")    pkg.name    = val;
            else if (key == "version") pkg.version = val;
            else if (key == "source")  pkg.source  = val;
        }
        else if (section == "files") {
            if (!waiting_md5) {
                cur_file.path = line;
                waiting_md5 = true;
            } else {
                cur_file.md5 = line;
                flush_file();
            }
        }
        else if (section == "services") {
            // "systemd:" → new init block
            if (line.back() == ':') {
                current_init = line.substr(0, line.size() - 1);
                pkg.services.push_back({current_init, {}});
            } else if (!current_init.empty()) {
                pkg.services.back().paths.push_back(line);
            }
        }
    }
    flush_file();

    if (pkg.name.empty())
        throw std::runtime_error("Invalid .nopile (missing name): " + path.string());

    return pkg;
}

void nopile_write(const fs::path& path, const NopileFile& pkg) {
    fs::create_directories(path.parent_path());
    std::ofstream f(path);
    if (!f) throw std::runtime_error("Cannot write: " + path.string());

    f << "name:    " << pkg.name    << '\n';
    f << "version: " << pkg.version << '\n';
    f << "source:  " << pkg.source  << '\n';

    f << "\n[files]\n";
    for (const auto& file : pkg.files) {
        f << file.path << '\n';
        f << file.md5  << '\n';
        f << '\n';
    }

    if (!pkg.services.empty()) {
        f << "[services]\n";
        for (const auto& svc : pkg.services) {
            f << svc.init_name << ":\n";
            for (const auto& p : svc.paths)
                f << p << '\n';
            f << '\n';
        }
    }
}
