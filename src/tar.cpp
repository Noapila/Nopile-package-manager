#include "nopile.hpp"

#include <fstream>
#include <sstream>
#include <iostream>
#include <cstdlib>

// Uses the system's tar binary — always present on any LFS
bool tar_extract(const fs::path& archive, const fs::path& dest) {
    fs::create_directories(dest);
    std::string cmd = "tar -xzf " + archive.string()
                    + " -C " + dest.string() + " 2>/dev/null";
    return std::system(cmd.c_str()) == 0;
}

// ── install.nopbuild runner ───────────────────────────────────────────────────
//
//  Supported instructions:
//    install <src> <dest>         copy file from package/ to dest on system
//    mkdir <path>                 create directory
//    chmod <mode> <path>          set permissions
//    symlink <target> <link>      create symlink
//
// ─────────────────────────────────────────────────────────────────────────────

static std::string trim(const std::string& s) {
    size_t b = s.find_first_not_of(" \t\r\n");
    if (b == std::string::npos) return {};
    size_t e = s.find_last_not_of(" \t\r\n");
    return s.substr(b, e - b + 1);
}

bool nopbuild_run(const fs::path& build_file, const fs::path& pkg_root) {
    std::ifstream f(build_file);
    if (!f) {
        std::cerr << "[nopile] cannot open: " << build_file << '\n';
        return false;
    }

    std::string line;
    int line_num = 0;
    bool ok = true;

    while (std::getline(f, line)) {
        ++line_num;
        line = trim(line);
        if (line.empty() || line[0] == '#') continue;

        std::istringstream ss(line);
        std::string cmd;
        ss >> cmd;

        if (cmd == "install") {
            std::string src, dest;
            ss >> src >> dest;
            if (src.empty() || dest.empty()) {
                std::cerr << "[nopile] nopbuild line " << line_num << ": bad install\n";
                ok = false; continue;
            }
            fs::path src_path = pkg_root / src;
            fs::path dst_path(dest);
            if (!fs::exists(src_path)) {
                std::cerr << "[nopile] not found: " << src_path << '\n';
                ok = false; continue;
            }
            fs::create_directories(dst_path.parent_path());
            fs::copy_file(src_path, dst_path,
                          fs::copy_options::overwrite_existing);
        }
        else if (cmd == "mkdir") {
            std::string path;
            ss >> path;
            if (path.empty()) { ok = false; continue; }
            fs::create_directories(fs::path(path));
        }
        else if (cmd == "chmod") {
            std::string mode, path;
            ss >> mode >> path;
            if (mode.empty() || path.empty()) { ok = false; continue; }
            std::string c = "chmod " + mode + " " + path;
            std::system(c.c_str());
        }
        else if (cmd == "symlink") {
            std::string target, link;
            ss >> target >> link;
            if (target.empty() || link.empty()) { ok = false; continue; }
            fs::path lp(link);
            fs::create_directories(lp.parent_path());
            if (fs::exists(lp) || fs::is_symlink(lp))
                fs::remove(lp);
            fs::create_symlink(fs::path(target), lp);
        }
        else {
            std::cerr << "[nopile] nopbuild line " << line_num
                      << ": unknown instruction: " << cmd << '\n';
        }
    }
    return ok;
}
