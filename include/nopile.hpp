#pragma once

#include <string>
#include <vector>
#include <map>
#include <filesystem>
#include <optional>

namespace fs = std::filesystem;

// ── Constants ─────────────────────────────────────────────────────────────────

inline constexpr const char* NOPILE_VERSION    = "0.1.0";
inline constexpr const char* NOPILE_PKG_DIR    = "/var/lib/nopile/packages";
inline constexpr const char* NOPILE_SOURCES_DIR= "/var/lib/nopile/sources";
inline constexpr const char* NOPILE_CACHE_DIR  = "/var/cache/nopile";
inline constexpr const char* NOPILE_DB         = "/var/lib/nopile/nopile.db";
inline constexpr const char* NOPILE_REPO_RAW   =
    "https://raw.githubusercontent.com/Noapila/Nopile-package-manager-repo/main";

// ── .nopile file ──────────────────────────────────────────────────────────────

struct InstalledFile {
    std::string path;
    std::string md5;
};

struct ServiceEntry {
    std::string init_name;           // "systemd", "runit", ...
    std::vector<std::string> paths;  // chemins installés pour cet init
};

struct NopileFile {
    std::string name;
    std::string version;
    std::string source;              // URL d'origine
    std::vector<InstalledFile> files;
    std::vector<ServiceEntry>  services;
};

// ── nopile.db entry ───────────────────────────────────────────────────────────

struct DbEntry {
    std::string name;
    std::string version;
    std::string binary_url;
    std::string binary_md5;
    std::string source_url;
    std::string source_md5;
};

// ── Functions ─────────────────────────────────────────────────────────────────

// nopile.db
std::vector<DbEntry> db_load();
std::optional<DbEntry> db_find(const std::string& name);
bool db_download();

// .nopile read/write
NopileFile nopile_parse(const fs::path& path);
void       nopile_write(const fs::path& path, const NopileFile& pkg);
fs::path   nopile_path(const std::string& name);
bool       nopile_exists(const std::string& name);

// network (no external deps — raw sockets + OpenSSL)
bool net_download(const std::string& url, const fs::path& dest);

// md5
std::string md5_file(const fs::path& path);

// tarball
bool tar_extract(const fs::path& archive, const fs::path& dest);

// nopbuild
bool nopbuild_run(const fs::path& build_file, const fs::path& pkg_root);

// commands
int cmd_install(const std::string& name, bool compile, bool local, const fs::path& local_path);
int cmd_remove (const std::string& name);
int cmd_update ();
int cmd_search (const std::string& query);
int cmd_help   ();
