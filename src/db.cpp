#include "nopile.hpp"

#include <fstream>
#include <sstream>
#include <iostream>

// ── nopile.db format ──────────────────────────────────────────────────────────
//
//  package: bash
//  version: 5.2.21
//  binary:  https://raw.githubusercontent.com/.../packages/binary/bash.tar.gz
//  binary-md5: d41d8cd98f00b204e9800998ecf8427e
//  source:  https://raw.githubusercontent.com/.../packages/source/bash.tar.gz
//  source-md5: a3f5b2c1...
//
//  package: vim
//  ...
// ─────────────────────────────────────────────────────────────────────────────

static std::string trim(const std::string& s) {
    size_t b = s.find_first_not_of(" \t\r\n");
    if (b == std::string::npos) return {};
    size_t e = s.find_last_not_of(" \t\r\n");
    return s.substr(b, e - b + 1);
}

bool db_download() {
    std::string url = std::string(NOPILE_REPO_RAW) + "/nopile.db";
    fs::create_directories(fs::path(NOPILE_DB).parent_path());
    return net_download(url, fs::path(NOPILE_DB));
}

std::vector<DbEntry> db_load() {
    std::vector<DbEntry> entries;
    std::ifstream f(NOPILE_DB);
    if (!f) return entries;

    DbEntry cur;
    std::string line;

    auto flush = [&]() {
        if (!cur.name.empty()) {
            entries.push_back(cur);
            cur = DbEntry{};
        }
    };

    while (std::getline(f, line)) {
        line = trim(line);
        if (line.empty() || line[0] == '#') continue;

        auto colon = line.find(':');
        if (colon == std::string::npos) continue;

        std::string key = trim(line.substr(0, colon));
        std::string val = trim(line.substr(colon + 1));

        if      (key == "package")    { flush(); cur.name       = val; }
        else if (key == "version")    { cur.version    = val; }
        else if (key == "binary")     { cur.binary_url = val; }
        else if (key == "binary-md5") { cur.binary_md5 = val; }
        else if (key == "source")     { cur.source_url = val; }
        else if (key == "source-md5") { cur.source_md5 = val; }
    }
    flush();
    return entries;
}

std::optional<DbEntry> db_find(const std::string& name) {
    for (auto& e : db_load())
        if (e.name == name) return e;
    return std::nullopt;
}
