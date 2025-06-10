let accessToken: string | null = null;
let refreshUrl: string = "/api/access_token";

async function refreshAccessToken() {
  const response = await fetch(refreshUrl, {
    method: "POST",
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
    },
  });
  if (response.status === 200) {
    const respData = await response.json();
    if (respData.access_token) {
      accessToken = respData.access_token;
    } else if (respData.login_url) {
      window.location.href =
        respData.login_url + "?continue=" + window.location.href;
    }
  } else {
    // Errors will get handled gracefully elsewhere.
  }
}

export async function fetchAuthenticated(url: string, opts?: any) {
  if (accessToken === null) {
    // Refresh the access token before trying to make any requests
    await refreshAccessToken();
  }

  const method = opts?.method || "GET";
  let headers = {
    Authorization: `Bearer ${accessToken}`,
    ...opts?.headers,
  };
  let response = await fetch(url, {
    method,
    headers,
    body: opts?.body,
  });
  if (response.status === 401) {
    // If we are unauthenticated, the access token may have expired. Try to
    // refresh it and then try again.
    await refreshAccessToken();
    headers = {
      Authorization: `Bearer ${accessToken}`,
      ...opts?.headers,
    };
    response = await fetch(url, {
      method,
      headers,
      body: opts?.body,
    });
  }
  return response;
}
