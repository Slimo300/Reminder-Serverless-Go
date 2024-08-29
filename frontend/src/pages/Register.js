import React, { useState } from "react";
import { Navigate } from "react-router-dom";

import { createUser, verifyUser } from "../requests/Cognito";

const RegisterForm = ({ loggedIn }) => {
  const [username, setUsername] = useState("");
  const [phoneNumber, setPhoneNumber] = useState("");
  const [password, setPassword] = useState("");
  const [rpassword, setRPassword] = useState("");

  const [verificationCode, setVerificationCode] = useState("");

  const [redirect, setRedirect] = useState(false);
  const [showVerification, setShowVerification] = useState(false);

  const [message, setMessage] = useState("");

  if (loggedIn) return <Navigate to="/"/>

  const submitVerify = async (e) => {
    e.preventDefault();

    if (verificationCode.trim() === "") {
      setMessage("No code provided");
      return;
    }

    verifyUser(username, verificationCode, (err, result) => {
      if (err) {
        setMessage(err.message);
        return;
      }
      setRedirect(true);
    });
  }

  const submitRegister = async (e) => {
    e.preventDefault();

    if (username.trim() === "") {
      setMessage("Email can't be blank");
      return;
    }
    if (phoneNumber.trim() === "") {
      setMessage("Phone Number can't be blank");
      return
    }
    if (password.trim() === "") {
      setMessage("Password can't be blank");
      return;
    }
    if (password !== rpassword) {
      setMessage("Passwords don't match");
      return;
    }

    createUser(username, phoneNumber, password, (err, result) => {
      if (err) {
        console.log(err);
        return;
      }
      console.log(result);
    })

    setShowVerification(true);
  }

  if (redirect) {
    return <Navigate to={"/login?message="+message.replaceAll(" ", "+")}/>;
  }

  return (
    <div className="container mt-4 pt-4">
      <div className="mt-5 d-flex justify-content-center">
        <div className="mt-5 row">
          {showVerification
          ?<form onSubmit={submitVerify}>
            <div className="display-1 mb-4 text-center text-primary"> Register</div>
            <div className="mb-3 text-center text-danger">{message}</div>
            <div className="mb-3 text-center">
              <label className="form-label">Verification Code</label>
              <input type="text" className="form-control" onChange={(e) => setVerificationCode(e.target.value)}/>
            </div>
            <div className="text-center">
              <button type="submit" className="btn btn-primary text-center">Submit</button>
            </div>
          </form>
          :<form onSubmit={submitRegister}>
            <div className="display-1 mb-4 text-center text-primary"> Register</div>
            <div className="mb-3 text-center text-danger">{message}</div>
            <div className="mb-3 text-center">
              <label className="form-label">Username</label>
              <input type="text" className="form-control" onChange={(e) => setUsername(e.target.value)}/>
            </div>
            <div className="mb-3 text-center">
              <label className="form-label">Phone Number</label>
              <input type="text" className="form-control" onChange={(e) => setPhoneNumber(e.target.value)}/>
            </div>
            <div className="mb-3 text-center">
              <label className="form-label">Password</label>
              <input type="password" className="form-control" onChange={(e) => setPassword(e.target.value)}/>
            </div>
            <div className="mb-3 text-center">
              <label className="form-label">Repeat Password</label>
              <input type="password" className="form-control" onChange={(e) => setRPassword(e.target.value)}/>
            </div>
            <div className="text-center">
              <button type="submit" className="btn btn-primary text-center">Submit</button>
            </div>
            <div className="display-5 mt-4 text-center text-primary"><a href="/login">or Log in</a></div>
          </form>}
        </div>
      </div>
      </div>
    );
  }

export default RegisterForm;