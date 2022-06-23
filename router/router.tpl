{{ range $index, $element := . -}}
server {
  listen 80;
  server_name {{ $element.ServiceDomain }};

  client_max_body_size {{ $element.ServiceMaxBodySize }};

  location / {
    resolver 127.0.0.11;
    
    proxy_send_timeout {{ $element.ServiceProxyTimeout }};
    proxy_read_timeout {{ $element.ServiceProxyTimeout }};
    proxy_connect_timeout {{ $element.ServiceProxyTimeout }};

    proxy_pass http://{{ $element.ServiceName }}:{{ $element.ServicePort }}{{ $element.ServicePath }};
  }
}
{{ end -}}
