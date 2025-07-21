import { serve } from "bun";
import index from "./index.html";
import { getServerPorts } from "./config";

const { backendPort, webUIPort } = getServerPorts();

const server = serve({
  port: webUIPort,
  routes: {
    // Proxy API requests to the Go backend
    "/api/*": async (req) => {
      const url = new URL(req.url);
      const backendUrl = `http://localhost:${backendPort}${url.pathname}${url.search}`;
      
      try {
        const response = await fetch(backendUrl, {
          method: req.method,
          headers: req.headers,
          body: req.body,
        });
        
        return new Response(response.body, {
          status: response.status,
          statusText: response.statusText,
          headers: response.headers,
        });
      } catch (error) {
        console.error('Proxy error:', error);
        return new Response(JSON.stringify({ error: 'Backend server unavailable' }), {
          status: 503,
          headers: {
            'Content-Type': 'application/json'
          }
        });
      }
    },

    // Serve index.html for all unmatched routes.
    "/*": index,

    "/api/hello": {
      async GET(req) {
        return Response.json({
          message: "Hello, world!",
          method: "GET",
        });
      },
      async PUT(req) {
        return Response.json({
          message: "Hello, world!",
          method: "PUT",
        });
      },
    },

    "/api/hello/:name": async req => {
      const name = req.params.name;
      return Response.json({
        message: `Hello, ${name}!`,
      });
    },
  },

  development: process.env.NODE_ENV !== "production" && {
    // Enable browser hot reloading in development
    hmr: true,

    // Echo console logs from the browser to the server
    console: true,
  },
});

console.log(`🚀 Server running at ${server.url}`);
