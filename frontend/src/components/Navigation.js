import React  from 'react';
import { NavLink } from 'react-router-dom';

import { signOut } from '../requests/Cognito';

import logo from "../statics/images/logo.png";

const Navigation = ({ toggleProfile, loggedIn, setLoggedIn }) => {

    const logout = async () => {
        window.localStorage.removeItem("LOGGED_IN");
        setLoggedIn(false);
        signOut();
    };

    let menu;
    
    if (!loggedIn) {
        menu = (
            <div className="collapse navbar-collapse" id="navbarCollapse">
                <ul className="navbar-nav mr-auto"></ul>
                <NavLink className="nav-item nav-link" to="/login">Login</NavLink>
                <NavLink className="nav-item nav-link" to="/register">Register</NavLink>
            </div>
        );
    } else {
        menu = (
            <div className="collapse navbar-collapse" id="navbarCollapse">
                <ul className="navbar-nav mr-auto">
                    <button type='button' className="navbar-brand order-1 btn btn-dark text-primary" onClick={toggleProfile}>Profile</button>
                </ul>

                <NavLink className="nav-item nav-link" to="/login?logout=true" onClick={logout}>Logout</NavLink>
            </div>
        );
    }

    return (
        <nav className="navbar navbar-expand-md navbar-dark bg-dark mb-4">
            <NavLink className="navbar-brand" to="/" >
                <img src={logo} alt="Logo" width="150" height="55" className="d-inline-block align-text-top" />
            </NavLink>
            <div className="collapse navbar-collapse" id="navbarCollapse">
                {menu}
            </div>
        </nav>
    )
}

export default Navigation;