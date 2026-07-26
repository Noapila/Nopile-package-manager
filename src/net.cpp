#include "nopile.hpp"

#include <openssl/ssl.h>
#include <openssl/err.h>
#include <sys/socket.h>
#include <netdb.h>
#include <unistd.h>

#include <iostream>
#include <fstream>
#include <sstream>
#include <cstring>

// ── Parse URL ─────────────────────────────────────────────────────────────────
// Supports https:// only (GitHub raw always uses HTTPS)

struct URL {
    std::string host;
    std::string path;
    std::string port = "443";
};

static std::optional<URL> parse_url(const std::string& url) {
    if (url.substr(0, 8) != "https://") return std::nullopt;
    std::string rest = url.substr(8);
    auto slash = rest.find('/');
    URL u;
    u.host = (slash == std::string::npos) ? rest : rest.substr(0, slash);
    u.path = (slash == std::string::npos) ? "/" : rest.substr(slash);
    return u;
}

// ── TCP connect ───────────────────────────────────────────────────────────────

static int tcp_connect(const std::string& host, const std::string& port) {
    addrinfo hints{}, *res;
    hints.ai_family   = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;

    if (getaddrinfo(host.c_str(), port.c_str(), &hints, &res) != 0)
        return -1;

    int fd = -1;
    for (auto* r = res; r; r = r->ai_next) {
        fd = socket(r->ai_family, r->ai_socktype, r->ai_protocol);
        if (fd < 0) continue;
        if (connect(fd, r->ai_addr, r->ai_addrlen) == 0) break;
        close(fd);
        fd = -1;
    }
    freeaddrinfo(res);
    return fd;
}

// ── HTTPS GET → file ──────────────────────────────────────────────────────────

bool net_download(const std::string& url, const fs::path& dest) {
    auto u_opt = parse_url(url);
    if (!u_opt) {
        std::cerr << "[nopile] invalid URL: " << url << '\n';
        return false;
    }
    URL u = *u_opt;

    // TCP
    int fd = tcp_connect(u.host, u.port);
    if (fd < 0) {
        std::cerr << "[nopile] connection failed: " << u.host << '\n';
        return false;
    }

    // SSL
    SSL_CTX* ctx = SSL_CTX_new(TLS_client_method());
    SSL_CTX_set_verify(ctx, SSL_VERIFY_PEER, nullptr);
    SSL_CTX_set_default_verify_paths(ctx);

    SSL* ssl = SSL_new(ctx);
    SSL_set_fd(ssl, fd);
    SSL_set_tlsext_host_name(ssl, u.host.c_str());

    if (SSL_connect(ssl) != 1) {
        std::cerr << "[nopile] SSL handshake failed\n";
        SSL_free(ssl);
        SSL_CTX_free(ctx);
        close(fd);
        return false;
    }

    // HTTP/1.1 GET
    std::string req =
        "GET " + u.path + " HTTP/1.1\r\n"
        "Host: " + u.host + "\r\n"
        "User-Agent: nopile/" + NOPILE_VERSION + "\r\n"
        "Connection: close\r\n\r\n";

    SSL_write(ssl, req.c_str(), (int)req.size());

    // Read response
    std::string response;
    char buf[4096];
    int n;
    while ((n = SSL_read(ssl, buf, sizeof(buf))) > 0)
        response.append(buf, n);

    SSL_free(ssl);
    SSL_CTX_free(ctx);
    close(fd);

    // Handle redirect (301/302)
    if (response.substr(9, 3) == "301" || response.substr(9, 3) == "302") {
        auto loc = response.find("Location: ");
        if (loc != std::string::npos) {
            auto end = response.find("\r\n", loc + 10);
            std::string new_url = response.substr(loc + 10, end - loc - 10);
            return net_download(new_url, dest);
        }
    }

    // Check 200
    if (response.substr(9, 3) != "200") {
        std::cerr << "[nopile] HTTP error for: " << url << '\n';
        return false;
    }

    // Strip headers
    auto header_end = response.find("\r\n\r\n");
    if (header_end == std::string::npos) return false;
    std::string body = response.substr(header_end + 4);

    // Write to file
    fs::create_directories(dest.parent_path());
    std::ofstream f(dest, std::ios::binary);
    if (!f) return false;
    f.write(body.data(), (std::streamsize)body.size());
    return f.good();
}
