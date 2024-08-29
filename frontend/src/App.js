import React, {useState} from "react";
import { BrowserRouter as Router, Route, Routes} from "react-router-dom";

import Navigation from "./components/Navigation";

import Main from "./pages/Main";
import RegisterForm from "./pages/Register";
import Page404 from "./pages/404";
import SignInForm from './pages/Login';

function App() {
  
  const [profileShow, setProfileShow] = useState(false);
  const toggleProfileShow = () => {
    setProfileShow(!profileShow);
  }

  const [loggedIn, setLoggedIn] = useState(window.localStorage.getItem("LOGGED_IN")?true:false);

  return (
      <div >
        <Router>
          <Navigation toggleProfile={toggleProfileShow} loggedIn={loggedIn} setLoggedIn={setLoggedIn} />
          <main>
            <Routes>
              <Route path="/" element={<Main profileShow={profileShow} toggleProfile={toggleProfileShow} loggedIn={loggedIn} />}/>
              <Route path="/login" element={<SignInForm loggedIn={loggedIn} setLoggedIn={setLoggedIn} />}/>
              <Route path="/register" element={<RegisterForm  loggedIn={loggedIn} />}/>
              <Route path="*" element={<Page404 />}/>
            </Routes>
          </main>
        </Router>
      </div>
  )
}

export default App;