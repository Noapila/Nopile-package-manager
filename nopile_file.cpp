#include "nopile.hpp"
#include <fstream>
#include <sstream>
#include <stdexcept>

static std::string trim(const std::string& s) {
    size_t b = s.find_first_not_of(" \t\r\n");
    if (b == std::string::npos) return {};
    size_t e = s.find_last_not_of(" \t\r\n");
    return s.substr(b, e - b + 1);
}

static std::vector<std::string> split_csv(const std::string& s) {
    std::vector<std::string> out;
    std::istringstream ss(s);
    std::string tok;
    while (std::getline(ss, tok, ','))
        out.push_back(trim(tok));
    return out;
}

PackageMeta parse_nopile_file(const fs::path& path) {
    std::ifstream f(path);
    if (!f) throw std::runtime_error("Cannot open: " + path.string());

    PackageMeta meta;
    std::string section, line;

    while (std::getline(f, line)) {
        line = trim(line);
        if (line.empty() || line[0] == '#') continue;

        if (line.front() == '[' && line.back() == ']') {
            section = line.substr(1, line.size() - 2);
            continue;
        }

        if (section == "package" || section == "source") {
            auto eq = line.find('=');
            if (eq == std::string::npos) continue;
            std::string key = trim(line.substr(0, eq));
            std::string val = trim(line.substr(eq + 1));

            if      (key == "name")             meta.name             = val;
            else if (key == "version")          meta.version          = val;
            else if (key == "description")      meta.description      = val;
            else if (key == "maintainer")       meta.maintainer       = val;
            else if (key == "arch")             meta.arch             = val;
            else if (key == "depends")          meta.depends          = split_csv(val);
            else if (key == "url")              meta.source_url       = val;
            else if (key == "build_system")     meta.build_system     = val;
            else if (key == "configure_flags")  meta.configure_flags  = val;
        }
        else if (section == "files") {
            if (!line.empty()) meta.files.push_back(line);
        }
    }

    if (meta.name.empty())
        throw std::runtime_error("Invalid .nopile file (missing name): " + path.string());

    return meta;
}

void write_nopile_file(const fs::path& path, const PackageMeta& meta) {
    std::ofstream f(path);
    if (!f) throw std::runtime_error("Cannot write: " + path.string());

    f << "[package]\n";
    f << "name        = " << meta.name        << '\n';
    f << "version     = " << meta.version     << '\n';
    f << "description = " << meta.description << '\n';
    f << "maintainer  = " << meta.maintainer  << '\n';
    f << "arch        = " << meta.arch        << '\n';

    if (!meta.depends.empty()) {
        f << "depends     = ";
        for (size_t i = 0; i < meta.depends.size(); ++i) {
            if (i) f << ", ";
            f << meta.depends[i];
        }
        f << '\n';
    }

    f << "\n[files]\n";
    for (const auto& file : meta.files)
        f << file << '\n';

    if (!meta.source_url.empty()) {
        f << "\n[source]\n";
        f << "url             = " << meta.source_url      << '\n';
        f << "build_system    = " << meta.build_system    << '\n';
        f << "configure_flags = " << meta.configure_flags << '\n';
    }
}
