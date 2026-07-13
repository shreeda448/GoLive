import { useState, useEffect } from "react";
import "./App.css";

function Dashboard({ deployID, curStatus, logs }) {
  if (!deployID) {
    return null;
  }
  let statusColor = "#24eaf7";
  if (curStatus === "BUILDING") statusColor = "#eab308";
  if (curStatus === "SUCCESS") statusColor = "#22c55e";
  if (curStatus === "FAILED") statusColor = "#ef4444";
  return (
    <>
      <div className="deployment-job card">
        <h3 id="deployment-job-title">Dashboard for deployment job</h3>
        <div className="status-grid">
          <div className="grid-cell label top-left">Deployment ID</div>
          <div className="grid-cell value top-right">{deployID}</div>
          <div className="grid-cell label bottom-left">Status</div>
          <div
            className="grid-cell value bottom-right"
            style={{ color: statusColor, fontWeight: "bold" }}
          >
            {curStatus}
          </div>
        </div>
      </div>
      <pre className="logs">
        {logs.map((log, index) => {
          return (
            <p className="log" key={index}>
              {log}
            </p>
          );
        })}
      </pre>
    </>
  );
}

function Title() {
  return <h1 id="title">GoLive Deploy Engine</h1>;
}

function NewDeploymentSection({ curURL, updateInput, fetchData, className }) {
  return (
    <div className={className}>
      <input
        id="projectURL"
        placeholder="Paste your  project github url"
        value={curURL}
        onChange={updateInput}
      />
      <button type="submit" id="deploy-button" onClick={fetchData}>
        Deploy
      </button>
    </div>
  );
}

function App() {
  const [curURL, SetCurURL] = useState("");
  const [deployID, SetDeployID] = useState("");
  const [curStatus, SetCurStatus] = useState("");
  const [deployButtonClicked, SetDeployButtonClicked] = useState(false);
  const [logs, SetLogs] = useState([]);
  const updateInput = (event) => {
    SetCurURL(event.target.value);
  };
  useEffect(() => {
    if (!deployID) {
      return;
    }
    const ws = new WebSocket(`ws://localhost:8080/logs?id=${deployID}`);
    ws.onmessage = (event) => {
      SetLogs((prevLogs) => [...prevLogs, event.data]);
    };
    const intervalID = setInterval(async () => {
      const res = await fetch(`http://localhost:8080/status?id=${deployID}`, {
        method: "GET",
        headers: { "Content-Type": "application/json" },
      });
      const curStat = await res.json();
      SetCurStatus(curStat.status);
      if (curStat.status === "SUCCESS" || curStat.status === "FAILED") {
        clearInterval(intervalID);
      }
    }, 2000);
    return () => {
      clearInterval(intervalID);
      ws.close();
    };
  }, [deployID]);
  const fetchData = async () => {
    const response = await fetch("http://localhost:8080/deploy", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ repo_url: curURL }),
    });
    if (!response.ok) {
      console.log("something went wrong");
      return;
    }
    const data = await response.json();
    SetDeployID(data.deploy_id);
    SetCurStatus(data.status);
    SetDeployButtonClicked(true);
    console.log(data.deploy_id);
    console.log(data.status);
  };
  return (
    <div className="app">
      <Title />
      <NewDeploymentSection
        curURL={curURL}
        updateInput={updateInput}
        fetchData={fetchData}
        className="newDeploymentSection"
      />
      {deployButtonClicked ? (
        <Dashboard
          logs={logs}
          deployID={deployID}
          curStatus={curStatus}
        ></Dashboard>
      ) : null}
    </div>
  );
}

export default App;
